package fsutil

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	cryptMagic      = "LUMENC1\n"
	cryptVersion    = 1
	cryptHeaderSize = 56
	cryptChunkSize  = 1 << 20
	cryptIterations = 210000
	cryptSuffix     = ".lumen"
)

var ErrPassword = errors.New("the password is wrong or the file is damaged")

func Encrypt(path, password string) (string, error) {
	if len(password) < 4 {
		return "", errors.New("choose a password with at least 4 characters")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("links cannot be encrypted")
	}
	if !info.Mode().IsRegular() && !info.IsDir() {
		return "", errors.New("only files and folders can be encrypted")
	}
	if info.IsDir() && Protected(path) {
		return "", ErrProtected
	}
	src := path
	var flags byte
	var size int64
	if info.IsDir() {
		tmp, err := os.CreateTemp(filepath.Dir(path), ".lumen-pack-*")
		if err != nil {
			return "", err
		}
		tmpName := tmp.Name()
		tmp.Close()
		defer os.Remove(tmpName)
		if err := packDir(path, tmpName); err != nil {
			return "", err
		}
		st, err := os.Stat(tmpName)
		if err != nil {
			return "", err
		}
		src = tmpName
		size = st.Size()
		flags |= 1
	} else {
		size = info.Size()
	}
	dir := filepath.Dir(path)
	out := filepath.Join(dir, UniqueName(dir, info.Name()+cryptSuffix))
	f, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	err = encryptStream(f, src, password, size, flags)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(out)
		return "", err
	}
	return out, nil
}

func Decrypt(path, password string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("links cannot be decrypted")
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("choose an encrypted file")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	header := make([]byte, cryptHeaderSize)
	if _, err := io.ReadFull(f, header); err != nil {
		return "", errors.New("this is not a Lumen encrypted file")
	}
	if err := checkHeader(header); err != nil {
		return "", err
	}
	flags := header[9]
	size := int64(binary.BigEndian.Uint64(header[48:56]))
	dir := filepath.Dir(path)
	base := strings.TrimSuffix(filepath.Base(path), cryptSuffix)
	if base == filepath.Base(path) || base == "" {
		base = filepath.Base(path) + ".decrypted"
	}
	if flags&1 != 0 {
		lower := strings.ToLower(base)
		for _, suffix := range []string{".tar.gz", ".tgz", ".tar"} {
			if strings.HasSuffix(lower, suffix) && len(base) > len(suffix) {
				base = base[:len(base)-len(suffix)]
				break
			}
		}
		if strings.TrimSpace(base) == "" {
			base = "restored"
		}
		tmp, err := os.MkdirTemp(dir, ".lumen-dec-*")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(tmp)
		archive := filepath.Join(tmp, base+".tar.gz")
		out, err := os.OpenFile(archive, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return "", err
		}
		err = decryptStream(out, f, password, header, size)
		if closeErr := out.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return "", err
		}
		return Extract(archive, dir)
	}
	target := filepath.Join(dir, UniqueName(dir, base))
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	err = decryptStream(out, f, password, header, size)
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(target)
		return "", err
	}
	return target, nil
}

func packDir(dir, out string) error {
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == dir {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		link := ""
		if info.Mode()&os.ModeSymlink != 0 {
			if link, err = os.Readlink(path); err != nil {
				return err
			}
		}
		if !info.Mode().IsRegular() && !info.IsDir() && link == "" {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)
		if info.IsDir() {
			hdr.Name += "/"
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(tw, src)
		src.Close()
		return err
	})
	if err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	return f.Close()
}

func IsEncrypted(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	head := make([]byte, len(cryptMagic))
	if _, err := io.ReadFull(f, head); err != nil {
		return false
	}
	return string(head) == cryptMagic
}

func encryptStream(dst io.Writer, src, password string, size int64, flags byte) error {
	salt := make([]byte, 16)
	nonceBase := make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return err
	}
	if _, err := io.ReadFull(rand.Reader, nonceBase); err != nil {
		return err
	}
	header := make([]byte, cryptHeaderSize)
	copy(header[0:8], cryptMagic)
	header[8] = cryptVersion
	header[9] = flags
	copy(header[16:32], salt)
	binary.BigEndian.PutUint32(header[32:36], cryptIterations)
	binary.BigEndian.PutUint32(header[36:40], cryptChunkSize)
	copy(header[40:48], nonceBase)
	binary.BigEndian.PutUint64(header[48:56], uint64(size))
	key := pbkdf2([]byte(password), salt, cryptIterations, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	w := bufio.NewWriterSize(dst, 256<<10)
	if _, err := w.Write(header); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	plain := make([]byte, cryptChunkSize)
	nonce := make([]byte, 12)
	copy(nonce, nonceBase)
	aad := make([]byte, cryptHeaderSize+4)
	copy(aad, header)
	var total int64
	var counter uint32
	for {
		n, readErr := io.ReadFull(in, plain)
		if readErr == io.EOF {
			break
		}
		if readErr != nil && readErr != io.ErrUnexpectedEOF {
			return readErr
		}
		if n == 0 {
			break
		}
		binary.BigEndian.PutUint32(nonce[8:], counter)
		binary.BigEndian.PutUint32(aad[cryptHeaderSize:], counter)
		sealed := gcm.Seal(nil, nonce, plain[:n], aad)
		var length [4]byte
		binary.BigEndian.PutUint32(length[:], uint32(len(sealed)))
		if _, err := w.Write(length[:]); err != nil {
			return err
		}
		if _, err := w.Write(sealed); err != nil {
			return err
		}
		total += int64(n)
		counter++
		if readErr == io.ErrUnexpectedEOF {
			break
		}
	}
	if total != size {
		return errors.New("the file changed while it was being encrypted")
	}
	return w.Flush()
}

func decryptStream(dst io.Writer, src io.Reader, password string, header []byte, size int64) error {
	flags := header[9]
	if flags&^byte(1) != 0 {
		return errors.New("this encrypted file uses an unknown format")
	}
	iterations := int(binary.BigEndian.Uint32(header[32:36]))
	chunkSize := int(binary.BigEndian.Uint32(header[36:40]))
	key := pbkdf2([]byte(password), header[16:32], iterations, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonce := make([]byte, 12)
	copy(nonce, header[40:48])
	aad := make([]byte, cryptHeaderSize+4)
	copy(aad, header)
	br := bufio.NewReaderSize(src, 256<<10)
	w := bufio.NewWriterSize(dst, 256<<10)
	var total int64
	var counter uint32
	for {
		var length [4]byte
		_, err := io.ReadFull(br, length[:])
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.New("the encrypted file is incomplete")
		}
		n := int(binary.BigEndian.Uint32(length[:]))
		if n <= gcm.Overhead() || n > chunkSize+gcm.Overhead() {
			return errors.New("the encrypted file is damaged")
		}
		sealed := make([]byte, n)
		if _, err := io.ReadFull(br, sealed); err != nil {
			return errors.New("the encrypted file is incomplete")
		}
		binary.BigEndian.PutUint32(nonce[8:], counter)
		binary.BigEndian.PutUint32(aad[cryptHeaderSize:], counter)
		plain, err := gcm.Open(nil, nonce, sealed, aad)
		if err != nil {
			return ErrPassword
		}
		total += int64(len(plain))
		if total > size {
			return errors.New("the encrypted file is damaged")
		}
		if _, err := w.Write(plain); err != nil {
			return err
		}
		counter++
	}
	if total != size {
		return errors.New("the encrypted file is incomplete")
	}
	return w.Flush()
}

func checkHeader(header []byte) error {
	if string(header[0:8]) != cryptMagic {
		return errors.New("this is not a Lumen encrypted file")
	}
	if header[8] != cryptVersion {
		return errors.New("this encrypted file needs a newer version of Lumen")
	}
	if header[9]&^byte(1) != 0 {
		return errors.New("this encrypted file uses an unknown format")
	}
	for _, b := range header[10:16] {
		if b != 0 {
			return errors.New("this encrypted file uses an unknown format")
		}
	}
	iterations := binary.BigEndian.Uint32(header[32:36])
	chunk := binary.BigEndian.Uint32(header[36:40])
	if iterations < 10000 || iterations > 2000000 || chunk < 64<<10 || chunk > 16<<20 {
		return errors.New("this encrypted file uses an unknown format")
	}
	if bytes.Equal(header[40:48], make([]byte, 8)) {
		return errors.New("this encrypted file uses an unknown format")
	}
	return nil
}

func pbkdf2(password, salt []byte, iterations, length int) []byte {
	mac := hmac.New(sha256.New, password)
	blocks := (length + mac.Size() - 1) / mac.Size()
	out := make([]byte, 0, blocks*mac.Size())
	var index [4]byte
	for block := 1; block <= blocks; block++ {
		mac.Reset()
		mac.Write(salt)
		binary.BigEndian.PutUint32(index[:], uint32(block))
		mac.Write(index[:])
		u := mac.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)
		for n := 1; n < iterations; n++ {
			mac.Reset()
			mac.Write(u)
			u = mac.Sum(u[:0])
			for i := range t {
				t[i] ^= u[i]
			}
		}
		out = append(out, t...)
	}
	return out[:length]
}
