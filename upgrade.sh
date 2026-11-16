#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
if [ ! -f go.mod ] || [ ! -d internal/server ] || [ ! -f web/js/app.js ]; then
  echo "run this script inside the Lumen project folder" >&2
  exit 1
fi
if ! command -v go >/dev/null 2>&1; then
  echo "Go is not installed." >&2
  exit 1
fi
if ! command -v python3 >/dev/null 2>&1; then
  echo "Python 3 is not installed." >&2
  exit 1
fi

mkdir -p 'internal/icons'
cat > 'internal/icons/icons.go' <<'LUMEN_FILE_0'
package icons

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type candidate struct {
	path string
	size int
	rank int
}

type Resolver struct {
	once  sync.Once
	items map[string][]candidate
}

func New() *Resolver {
	return &Resolver{}
}

func ValidName(name string) bool {
	if name == "" || len(name) > 128 || strings.HasPrefix(name, ".") {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.', r == '+':
		default:
			return false
		}
	}
	return true
}

func (r *Resolver) Lookup(name string, size int) (string, string, bool) {
	if !ValidName(name) {
		return "", "", false
	}
	r.once.Do(r.build)
	if size < 8 {
		size = 8
	}
	if size > 1024 {
		size = 1024
	}
	items := r.items[name]
	if len(items) == 0 {
		items = r.items[strings.ToLower(name)]
	}
	if len(items) == 0 {
		return "", "", false
	}
	best := items[0]
	bestScore := 1 << 30
	for _, item := range items {
		score := item.rank*4096 + abs(item.size-size)
		if score < bestScore {
			best = item
			bestScore = score
		}
	}
	ext := strings.ToLower(filepath.Ext(best.path))
	base := strings.TrimSuffix(filepath.Base(best.path), filepath.Ext(best.path))
	if !ValidName(base) {
		return "", "", false
	}
	typ := "image/png"
	if ext == ".svg" {
		typ = "image/svg+xml"
	}
	return best.path, typ, true
}

func (r *Resolver) build() {
	r.items = map[string][]candidate{}
	for _, root := range iconRoots() {
		ents, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, ent := range ents {
			if !ent.IsDir() {
				continue
			}
			rank := themeRank(ent.Name())
			if rank < 0 {
				continue
			}
			r.walk(root, filepath.Join(root, ent.Name()), rank)
		}
	}
	for _, root := range pixmapRoots() {
		r.walk(root, root, 40)
	}
	for name := range r.items {
		sort.Slice(r.items[name], func(i, j int) bool {
			a, b := r.items[name][i], r.items[name][j]
			if a.rank != b.rank {
				return a.rank < b.rank
			}
			return a.size < b.size
		})
	}
}

func (r *Resolver) walk(root, dir string, rank int) {
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			lower := strings.ToLower(d.Name())
			if lower == "cursors" || lower == ".git" || strings.HasSuffix(lower, "cursors") {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".svg" && ext != ".png" {
			return nil
		}
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		base = strings.TrimSuffix(base, "@2x")
		if !ValidName(base) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		size := 0
		for _, seg := range strings.Split(rel, string(os.PathSeparator)) {
			if n := segmentSize(seg); n > 0 {
				size = n
			}
		}
		if size == 0 {
			if ext == ".svg" {
				size = 512
			} else {
				size = 64
			}
		}
		r.items[base] = append(r.items[base], candidate{path: path, size: size, rank: rank})
		return nil
	})
}

func segmentSize(seg string) int {
	lower := strings.ToLower(seg)
	if lower == "scalable" || lower == "symbolic" {
		return 512
	}
	lower = strings.TrimSuffix(lower, "@2x")
	lower = strings.TrimSuffix(lower, "x")
	parts := strings.Split(lower, "x")
	if len(parts) == 0 {
		return 0
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil || n < 8 || n > 1024 {
		return 0
	}
	if len(parts) > 1 && parts[1] != parts[0] {
		return n
	}
	return n
}

func iconRoots() []string {
	var roots []string
	seen := map[string]bool{}
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		roots = append(roots, p)
	}
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		home, _ := os.UserHomeDir()
		data = filepath.Join(home, ".local", "share")
	}
	add(filepath.Join(data, "icons"))
	if home, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(home, ".icons"))
	}
	system := os.Getenv("XDG_DATA_DIRS")
	if system == "" {
		system = "/usr/local/share:/usr/share"
	}
	for _, d := range strings.Split(system, ":") {
		if d != "" {
			add(filepath.Join(d, "icons"))
		}
	}
	return roots
}

func pixmapRoots() []string {
	var roots []string
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, filepath.Join(home, ".local", "share", "pixmaps"))
	}
	roots = append(roots, "/usr/local/share/pixmaps", "/usr/share/pixmaps")
	return roots
}

func themeRank(theme string) int {
	lower := strings.ToLower(theme)
	switch lower {
	case "breeze-dark":
		return 0
	case "breeze":
		return 1
	case "papirus-dark":
		return 2
	case "papirus":
		return 3
	case "adwaita":
		return 4
	case "hicolor":
		return 5
	case "oxygen":
		return 6
	case "gnome":
		return 7
	case "default":
		return 8
	}
	switch {
	case strings.Contains(lower, "breeze"):
		return 10
	case strings.Contains(lower, "papirus"):
		return 11
	case strings.Contains(lower, "adwaita"):
		return 12
	case strings.Contains(lower, "oxygen"):
		return 13
	}
	return -1
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
LUMEN_FILE_0

mkdir -p 'internal/fsutil'
cat > 'internal/fsutil/crypt.go' <<'LUMEN_FILE_1'
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
LUMEN_FILE_1

mkdir -p 'internal/dev'
cat > 'internal/dev/dev.go' <<'LUMEN_FILE_2'
package dev

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type MissingError struct {
	Tool string
	Hint string
}

var ErrUnsupported = errors.New("this file type cannot be compiled or run")

func (e *MissingError) Error() string {
	if e.Hint != "" {
		return e.Tool + " is not installed, install it with: " + e.Hint
	}
	return e.Tool + " is not installed"
}

type Info struct {
	Supported bool   `json:"supported"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Command   string `json:"command,omitempty"`
	Hint      string `json:"hint,omitempty"`
}

type Result struct {
	OK        bool   `json:"ok"`
	ID        string `json:"id"`
	Language  string `json:"language"`
	Command   string `json:"command"`
	Build     string `json:"build,omitempty"`
	Output    string `json:"output"`
	Exit      int    `json:"exit"`
	Duration  int64  `json:"durationMs"`
	TimedOut  bool   `json:"timedOut,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}

type kind struct {
	id      string
	name    string
	build   bool
	command func(src, work string) (build [][]string, run []string)
	hint    string
}

var kinds = map[string]kind{
	"py":  {"python", "Python", false, func(src, work string) ([][]string, []string) { return nil, []string{"python3", src} }, "sudo apt install python3"},
	"pyw": {"python", "Python", false, func(src, work string) ([][]string, []string) { return nil, []string{"python3", src} }, "sudo apt install python3"},
	"c": {"c", "C", true, func(src, work string) ([][]string, []string) {
		return [][]string{{"cc", "-O2", "-pipe", "-o", filepath.Join(work, "app"), src, "-lm"}}, []string{filepath.Join(work, "app")}
	}, "sudo apt install build-essential"},
	"h": {"c", "C header", false, nil, "choose a .c file to compile"},
	"cpp": {"cpp", "C++", true, func(src, work string) ([][]string, []string) {
		return [][]string{{"c++", "-O2", "-pipe", "-std=c++17", "-o", filepath.Join(work, "app"), src}}, []string{filepath.Join(work, "app")}
	}, "sudo apt install build-essential"},
	"cc": {"cpp", "C++", true, func(src, work string) ([][]string, []string) {
		return [][]string{{"c++", "-O2", "-pipe", "-std=c++17", "-o", filepath.Join(work, "app"), src}}, []string{filepath.Join(work, "app")}
	}, "sudo apt install build-essential"},
	"cxx": {"cpp", "C++", true, func(src, work string) ([][]string, []string) {
		return [][]string{{"c++", "-O2", "-pipe", "-std=c++17", "-o", filepath.Join(work, "app"), src}}, []string{filepath.Join(work, "app")}
	}, "sudo apt install build-essential"},
	"go": {"go", "Go", false, func(src, work string) ([][]string, []string) { return nil, []string{"go", "run", src} }, "sudo apt install golang-go"},
	"rs": {"rust", "Rust", true, func(src, work string) ([][]string, []string) {
		return [][]string{{"rustc", "-O", "-o", filepath.Join(work, "app"), src}}, []string{filepath.Join(work, "app")}
	}, "sudo apt install rustc"},
	"java": {"java", "Java", true, func(src, work string) ([][]string, []string) {
		return [][]string{{"javac", "-encoding", "UTF-8", "-d", work, src}}, []string{"java", "-cp", work, javaClass(src)}
	}, "sudo apt install default-jdk"},
	"kt": {"kotlin", "Kotlin", true, func(src, work string) ([][]string, []string) {
		out := filepath.Join(work, "app.jar")
		return [][]string{{"kotlinc", src, "-include-runtime", "-d", out}}, []string{"java", "-jar", out}
	}, "sudo apt install kotlin"},
	"js":    {"javascript", "JavaScript", false, func(src, work string) ([][]string, []string) { return nil, []string{"node", src} }, "sudo apt install nodejs"},
	"mjs":   {"javascript", "JavaScript", false, func(src, work string) ([][]string, []string) { return nil, []string{"node", src} }, "sudo apt install nodejs"},
	"cjs":   {"javascript", "JavaScript", false, func(src, work string) ([][]string, []string) { return nil, []string{"node", src} }, "sudo apt install nodejs"},
	"ts":    {"typescript", "TypeScript", false, tsRun, "sudo apt install deno"},
	"sh":    {"shell", "Shell", false, func(src, work string) ([][]string, []string) { return nil, []string{"bash", src} }, "sudo apt install bash"},
	"bash":  {"shell", "Bash", false, func(src, work string) ([][]string, []string) { return nil, []string{"bash", src} }, "sudo apt install bash"},
	"zsh":   {"shell", "Zsh", false, func(src, work string) ([][]string, []string) { return nil, []string{"zsh", src} }, "sudo apt install zsh"},
	"fish":  {"shell", "Fish", false, func(src, work string) ([][]string, []string) { return nil, []string{"fish", src} }, "sudo apt install fish"},
	"rb":    {"ruby", "Ruby", false, func(src, work string) ([][]string, []string) { return nil, []string{"ruby", src} }, "sudo apt install ruby"},
	"php":   {"php", "PHP", false, func(src, work string) ([][]string, []string) { return nil, []string{"php", src} }, "sudo apt install php-cli"},
	"pl":    {"perl", "Perl", false, func(src, work string) ([][]string, []string) { return nil, []string{"perl", src} }, "sudo apt install perl"},
	"lua":   {"lua", "Lua", false, func(src, work string) ([][]string, []string) { return nil, []string{"lua", src} }, "sudo apt install lua5.4"},
	"r":     {"r", "R", false, func(src, work string) ([][]string, []string) { return nil, []string{"Rscript", src} }, "sudo apt install r-base"},
	"swift": {"swift", "Swift", false, func(src, work string) ([][]string, []string) { return nil, []string{"swift", src} }, "sudo apt install swift"},
	"hs":    {"haskell", "Haskell", false, func(src, work string) ([][]string, []string) { return nil, []string{"runghc", src} }, "sudo apt install ghc"},
	"ex":    {"elixir", "Elixir", false, func(src, work string) ([][]string, []string) { return nil, []string{"elixir", src} }, "sudo apt install elixir"},
	"exs":   {"elixir", "Elixir", false, func(src, work string) ([][]string, []string) { return nil, []string{"elixir", src} }, "sudo apt install elixir"},
	"dart":  {"dart", "Dart", false, func(src, work string) ([][]string, []string) { return nil, []string{"dart", "run", src} }, "sudo apt install dart"},
	"jl":    {"julia", "Julia", false, func(src, work string) ([][]string, []string) { return nil, []string{"julia", src} }, "sudo apt install julia"},
	"s": {"asm", "Assembly", true, func(src, work string) ([][]string, []string) {
		return [][]string{{"cc", "-o", filepath.Join(work, "app"), src}}, []string{filepath.Join(work, "app")}
	}, "sudo apt install build-essential"},
	"asm": {"asm", "Assembly", true, func(src, work string) ([][]string, []string) {
		return [][]string{{"nasm", "-f", "elf64", "-o", filepath.Join(work, "app.o"), src}, {"cc", "-o", filepath.Join(work, "app"), filepath.Join(work, "app.o")}}, []string{filepath.Join(work, "app")}
	}, "sudo apt install nasm build-essential"},
	"cs":     {"csharp", "C#", true, csRun, "sudo apt install mono-mcs"},
	"coffee": {"coffee", "CoffeeScript", false, func(src, work string) ([][]string, []string) { return nil, []string{"coffee", src} }, "sudo apt install coffeescript"},
}

func Detect(path string) (kind, bool) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	if k, ok := kinds[ext]; ok && k.command != nil {
		return k, true
	}
	if k, ok := shebangKind(path); ok {
		return k, true
	}
	return kind{}, false
}

func Describe(path string) Info {
	k, ok := Detect(path)
	if !ok {
		return Info{}
	}
	_, run := k.command(path, "")
	return Info{Supported: true, ID: k.id, Name: k.name, Command: strings.Join(run, " ")}
}

func Run(ctx context.Context, path, argLine string) (Result, error) {
	k, ok := Detect(path)
	if !ok {
		return Result{}, ErrUnsupported
	}
	info, err := os.Stat(path)
	if err != nil {
		return Result{}, err
	}
	if !info.Mode().IsRegular() {
		return Result{}, errors.New("choose a regular file")
	}
	args, err := splitArgs(argLine)
	if err != nil {
		return Result{}, err
	}
	work, err := os.MkdirTemp("", "lumen-dev-*")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(work)
	build, run := k.command(path, work)
	for _, cmd := range append(build, run) {
		if len(cmd) == 0 {
			continue
		}
		if filepath.IsAbs(cmd[0]) || strings.ContainsRune(cmd[0], os.PathSeparator) {
			continue
		}
		if _, err := exec.LookPath(cmd[0]); err != nil {
			return Result{}, &MissingError{Tool: cmd[0], Hint: k.hint}
		}
	}
	dir := filepath.Dir(path)
	res := Result{ID: k.id, Language: k.name}
	started := time.Now()
	defer func() { res.Duration = time.Since(started).Milliseconds() }()
	if len(build) > 0 {
		res.Build = display(build[0])
	}
	var out limited
	for _, cmd := range build {
		code, timedOut := runCommand(ctx, dir, cmd, nil, &out)
		if timedOut || code != 0 {
			res.Output = out.String()
			res.Exit = code
			res.TimedOut = timedOut
			res.OK = false
			res.Duration = time.Since(started).Milliseconds()
			return res, nil
		}
	}
	full := append(append([]string{}, run...), args...)
	res.Command = display(full)
	code, timedOut := runCommand(ctx, dir, full, nil, &out)
	res.Output = out.String()
	res.Exit = code
	res.TimedOut = timedOut
	res.Truncated = out.truncated
	res.OK = code == 0 && !timedOut
	res.Duration = time.Since(started).Milliseconds()
	return res, nil
}

func runCommand(ctx context.Context, dir string, argv []string, stdin io.Reader, out *limited) (int, bool) {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Stdout = out
	cmd.Stderr = out
	if stdin != nil {
		cmd.Stdin = stdin
	}
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		out.Write([]byte("\nStopped after the time limit."))
		return -1, true
	}
	if err == nil {
		return 0, false
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), false
	}
	out.Write([]byte("\n" + err.Error()))
	return -1, false
}

func tsRun(src, work string) ([][]string, []string) {
	if _, err := exec.LookPath("deno"); err == nil {
		return nil, []string{"deno", "run", "--quiet", src}
	}
	if _, err := exec.LookPath("bun"); err == nil {
		return nil, []string{"bun", "run", src}
	}
	if _, err := exec.LookPath("ts-node"); err == nil {
		return nil, []string{"ts-node", src}
	}
	return nil, []string{"deno", "run", "--quiet", src}
}

func csRun(src, work string) ([][]string, []string) {
	exe := filepath.Join(work, "app.exe")
	if _, err := exec.LookPath("dotnet-script"); err == nil {
		return nil, []string{"dotnet-script", src}
	}
	if _, err := exec.LookPath("mcs"); err == nil {
		return [][]string{{"mcs", "-out:" + exe, src}}, []string{"mono", exe}
	}
	return [][]string{{"csc", "-out:" + exe, src}}, []string{"mono", exe}
}

func shebangKind(path string) (kind, bool) {
	f, err := os.Open(path)
	if err != nil {
		return kind{}, false
	}
	defer f.Close()
	buf := make([]byte, 256)
	n, _ := f.Read(buf)
	line := strings.TrimSpace(string(buf[:n]))
	if !strings.HasPrefix(line, "#!") {
		return kind{}, false
	}
	fields := strings.Fields(strings.TrimSpace(strings.TrimPrefix(line, "#!")))
	if len(fields) == 0 {
		return kind{}, false
	}
	name := filepath.Base(fields[0])
	if name == "env" {
		for _, f := range fields[1:] {
			if strings.HasPrefix(f, "-") {
				continue
			}
			name = filepath.Base(f)
			break
		}
	}
	switch name {
	case "python", "python3":
		return kinds["py"], true
	case "node", "nodejs":
		return kinds["js"], true
	case "bash", "sh":
		return kinds["bash"], true
	case "zsh":
		return kinds["zsh"], true
	case "fish":
		return kinds["fish"], true
	case "ruby":
		return kinds["rb"], true
	case "perl":
		return kinds["pl"], true
	case "php":
		return kinds["php"], true
	case "lua":
		return kinds["lua"], true
	case "rscript":
		return kinds["r"], true
	case "deno":
		return kinds["ts"], true
	case "bun":
		return kinds["ts"], true
	}
	return kind{}, false
}

func javaClass(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	data, err := os.ReadFile(path)
	if err != nil {
		return base
	}
	if len(data) > 1<<20 {
		data = data[:1<<20]
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if f != "class" && f != "interface" && f != "enum" && f != "record" {
				continue
			}
			if i+1 >= len(fields) {
				continue
			}
			name := strings.Trim(fields[i+1], "{<(")
			if name != "" {
				return name
			}
		}
	}
	return base
}

func splitArgs(line string) ([]string, error) {
	var out []string
	var cur strings.Builder
	quote := rune(0)
	escaped := false
	started := false
	for _, r := range line {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
			started = true
		case r == '\\' && quote != '\'':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
			started = true
		case r == '"' || r == '\'':
			quote = r
			started = true
		case r == ' ' || r == '\t' || r == '\n':
			if started {
				out = append(out, cur.String())
				cur.Reset()
				started = false
			}
		default:
			cur.WriteRune(r)
			started = true
		}
	}
	if escaped {
		return nil, errors.New("the arguments end with a backslash")
	}
	if quote != 0 {
		return nil, errors.New("a quote in the arguments is not closed")
	}
	if started {
		out = append(out, cur.String())
	}
	return out, nil
}

func display(argv []string) string {
	parts := make([]string, len(argv))
	for i, arg := range argv {
		if arg == "" || strings.ContainsAny(arg, " \t\n\"'\\$") {
			parts[i] = strconv.Quote(arg)
		} else {
			parts[i] = arg
		}
	}
	return strings.Join(parts, " ")
}

type limited struct {
	buf       bytes.Buffer
	truncated bool
}

func (w *limited) Write(p []byte) (int, error) {
	const max = 256 << 10
	left := max - w.buf.Len()
	if left > 0 {
		if len(p) > left {
			w.buf.Write(p[:left])
			w.truncated = true
		} else {
			w.buf.Write(p)
		}
	} else if len(p) > 0 {
		w.truncated = true
	}
	return len(p), nil
}

func (w *limited) String() string {
	if !w.truncated {
		return w.buf.String()
	}
	return w.buf.String() + "\n… output truncated"
}
LUMEN_FILE_2

mkdir -p 'internal/server'
cat > 'internal/server/handlers_icons.go' <<'LUMEN_FILE_3'
package server

import (
	"io"
	"net/http"
	"os"
	"strconv"
)

func (s *Server) themeIcon(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	size := 64
	if given, err := strconv.Atoi(r.URL.Query().Get("size")); err == nil && given > 0 && given <= 1024 {
		size = given
	}
	path, typ, ok := s.icons.Lookup(name, size)
	if !ok {
		writeError(w, http.StatusNotFound, "that icon is not available")
		return
	}
	f, err := os.Open(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "that icon is not available")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		writeError(w, http.StatusNotFound, "that icon is not available")
		return
	}
	h := w.Header()
	h.Set("Content-Type", typ)
	h.Set("Cache-Control", "private, max-age=86400")
	if typ == "image/svg+xml" {
		h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	}
	w.WriteHeader(http.StatusOK)
	io.Copy(w, io.LimitReader(f, 4<<20))
}
LUMEN_FILE_3

mkdir -p 'internal/server'
cat > 'internal/server/handlers_crypt.go' <<'LUMEN_FILE_4'
package server

import (
	"encoding/json"
	"net/http"

	"lumen/internal/fsutil"
)

func (s *Server) crypt(w http.ResponseWriter, r *http.Request) {
	if !s.writable(w) {
		return
	}
	var req struct {
		Path     string `json:"path"`
		Password string `json:"password"`
		Action   string `json:"action"`
		Trash    bool   `json:"trash"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "the request could not be read")
		return
	}
	path, err := s.cleanPath(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.Password) < 4 {
		writeError(w, http.StatusBadRequest, "choose a password with at least 4 characters")
		return
	}
	var out string
	var warning string
	switch req.Action {
	case "encrypt":
		out, err = fsutil.Encrypt(path, req.Password)
		if err == nil {
			s.opts.Index.Add(out)
			if req.Trash {
				if trashErr := fsutil.MoveToTrash(path); trashErr != nil {
					warning = "The encrypted copy was created, but the original could not be moved to the trash."
				} else {
					s.opts.Index.Remove(path)
				}
			}
		}
	case "decrypt":
		out, err = fsutil.Decrypt(path, req.Password)
		if err == nil {
			s.opts.Index.Add(out)
		}
	default:
		writeError(w, http.StatusBadRequest, "unknown action")
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "to": out, "warning": warning})
}
LUMEN_FILE_4

mkdir -p 'internal/server'
cat > 'internal/server/handlers_dev.go' <<'LUMEN_FILE_5'
package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"lumen/internal/dev"
)

func (s *Server) devInfo(w http.ResponseWriter, r *http.Request) {
	path, err := s.pathParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dev.Describe(path))
}

func (s *Server) devRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
		Args string `json:"args"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "the request could not be read")
		return
	}
	path, err := s.cleanPath(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 75*time.Second)
	defer cancel()
	res, err := dev.Run(ctx, path, req.Args)
	if err != nil {
		var missing *dev.MissingError
		if errors.As(err, &missing) {
			writeError(w, http.StatusUnprocessableEntity, missing.Error())
			return
		}
		if errors.Is(err, dev.ErrUnsupported) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
LUMEN_FILE_5

mkdir -p 'web/js'
cat > 'web/js/extras.js' <<'LUMEN_FILE_6'
(() => {
  'use strict';

  const api = {};
  let bound = false;

  const THEMES = [
    ['rose', 'Rose pink'],
    ['red', 'Muted red'],
    ['blue', 'Slate blue'],
    ['green', 'Sage green'],
    ['gray', 'Soft gray'],
  ];

  function themeValue(name) {
    return THEMES.some(([id]) => id === name) ? name : 'rose';
  }

  function savedTheme() {
    try {
      const raw = localStorage.getItem('lumen.theme');
      return themeValue(raw === null ? 'rose' : JSON.parse(raw));
    } catch {
      return 'rose';
    }
  }

  function applyTheme(name) {
    const id = themeValue(name);
    document.documentElement.dataset.theme = id;
    return id;
  }

  function saveTheme(name) {
    const id = applyTheme(name);
    try {
      localStorage.setItem('lumen.theme', JSON.stringify(id));
    } catch {}
    return id;
  }

  function themeOptions() {
    const current = savedTheme();
    return THEMES.map(([id, label]) => `<option value="${id}"${id === current ? ' selected' : ''}>${label}</option>`).join('');
  }

  applyTheme(savedTheme());

  window.addEventListener('storage', (ev) => {
    if (ev.key === 'lumen.theme') applyTheme(savedTheme());
  });

  const lazy = 'IntersectionObserver' in window ? new IntersectionObserver((entries) => {
    for (const entry of entries) {
      if (!entry.isIntersecting) continue;
      const img = entry.target;
      lazy.unobserve(img);
      if (!img.dataset.src) continue;
      img.src = img.dataset.src;
      delete img.dataset.src;
    }
  }, { rootMargin: '300px' }) : null;

  function watchIcons(root) {
    const nodes = [];
    if (root.nodeType === 1 && root.matches('.xicon-img[data-src]')) nodes.push(root);
    if (root.querySelectorAll) nodes.push(...root.querySelectorAll('.xicon-img[data-src]'));
    for (const img of nodes) {
      if (lazy) {
        lazy.observe(img);
      } else {
        img.src = img.dataset.src;
        delete img.dataset.src;
      }
    }
  }

  const EXT_ICONS = {
    pdf: 'application-pdf',
    doc: 'x-office-document',
    docx: 'x-office-document',
    odt: 'x-office-document',
    rtf: 'x-office-document',
    epub: 'application-epub+zip',
    mobi: 'application-epub+zip',
    djvu: 'image-x-generic',
    xls: 'x-office-spreadsheet',
    xlsx: 'x-office-spreadsheet',
    ods: 'x-office-spreadsheet',
    csv: 'x-office-spreadsheet',
    tsv: 'x-office-spreadsheet',
    ppt: 'x-office-presentation',
    pptx: 'x-office-presentation',
    odp: 'x-office-presentation',
    txt: 'text-x-generic',
    md: 'text-markdown',
    markdown: 'text-markdown',
    rst: 'text-x-generic',
    log: 'text-x-generic',
    nfo: 'text-x-generic',
    srt: 'text-x-generic',
    vtt: 'text-x-generic',
    tex: 'text-x-tex',
    json: 'application-json',
    yaml: 'application-x-yaml',
    yml: 'application-x-yaml',
    toml: 'application-x-toml',
    ini: 'text-x-generic',
    conf: 'text-x-generic',
    cfg: 'text-x-generic',
    env: 'text-x-generic',
    xml: 'application-xml',
    desktop: 'application-x-desktop',
    service: 'text-x-generic',
    py: 'text-x-python',
    pyw: 'text-x-python',
    c: 'text-x-csrc',
    h: 'text-x-chdr',
    cpp: 'text-x-c++src',
    cc: 'text-x-c++src',
    cxx: 'text-x-c++src',
    hpp: 'text-x-c++src',
    cs: 'text-x-csharp',
    rs: 'text-x-rust',
    java: 'text-x-java',
    kt: 'text-x-kotlin',
    go: 'text-x-go',
    rb: 'text-x-ruby',
    php: 'text-x-php',
    js: 'application-javascript',
    mjs: 'application-javascript',
    cjs: 'application-javascript',
    jsx: 'text-x-javascript',
    ts: 'text-x-typescript',
    tsx: 'text-x-typescript',
    vue: 'text-x-generic',
    svelte: 'text-x-generic',
    html: 'text-html',
    htm: 'text-html',
    css: 'text-css',
    scss: 'text-css',
    sass: 'text-css',
    sh: 'text-x-script',
    bash: 'text-x-script',
    zsh: 'text-x-script',
    fish: 'text-x-script',
    lua: 'text-x-lua',
    pl: 'text-x-perl',
    r: 'text-x-r',
    sql: 'text-x-sql',
    swift: 'text-x-swift',
    dart: 'text-x-dart',
    asm: 'text-x-asm',
    s: 'text-x-asm',
    cmake: 'text-x-cmake',
    mk: 'text-x-makefile',
    gradle: 'text-x-generic',
    jpg: 'image-x-generic',
    jpeg: 'image-x-generic',
    jpe: 'image-x-generic',
    jfif: 'image-x-generic',
    png: 'image-x-generic',
    gif: 'image-x-generic',
    webp: 'image-x-generic',
    bmp: 'image-x-generic',
    tif: 'image-x-generic',
    tiff: 'image-x-generic',
    svg: 'image-x-generic',
    avif: 'image-x-generic',
    heic: 'image-x-generic',
    heif: 'image-x-generic',
    ico: 'image-x-generic',
    jxl: 'image-x-generic',
    apng: 'image-x-generic',
    mp4: 'video-x-generic',
    mkv: 'video-x-generic',
    webm: 'video-x-generic',
    mov: 'video-x-generic',
    avi: 'video-x-generic',
    m4v: 'video-x-generic',
    flv: 'video-x-generic',
    wmv: 'video-x-generic',
    mpg: 'video-x-generic',
    mpeg: 'video-x-generic',
    m2ts: 'video-x-generic',
    '3gp': 'video-x-generic',
    ogv: 'video-x-generic',
    vob: 'video-x-generic',
    mp3: 'audio-x-generic',
    flac: 'audio-x-generic',
    wav: 'audio-x-generic',
    ogg: 'audio-x-generic',
    oga: 'audio-x-generic',
    opus: 'audio-x-generic',
    m4a: 'audio-x-generic',
    aac: 'audio-x-generic',
    wma: 'audio-x-generic',
    aiff: 'audio-x-generic',
    mka: 'audio-x-generic',
    amr: 'audio-x-generic',
    zip: 'application-zip',
    tar: 'application-x-tar',
    gz: 'application-x-compressed-tar',
    tgz: 'application-x-compressed-tar',
    bz2: 'application-x-bzip-compressed-tar',
    tbz: 'application-x-bzip-compressed-tar',
    tbz2: 'application-x-bzip-compressed-tar',
    xz: 'application-x-xz-compressed-tar',
    txz: 'application-x-xz-compressed-tar',
    zst: 'application-zstd',
    '7z': 'application-x-7z-compressed',
    rar: 'application-x-rar',
    iso: 'application-x-cd-image',
    img: 'application-x-cd-image',
    deb: 'application-x-deb',
    rpm: 'application-x-rpm',
    appimage: 'application-x-executable',
    jar: 'application-java-archive',
    cab: 'application-x-cab',
    lz4: 'application-x-lz4',
    ttf: 'font-x-generic',
    otf: 'font-x-generic',
    woff: 'font-x-generic',
    woff2: 'font-x-generic',
    exe: 'application-x-executable',
    msi: 'application-x-msi',
    bin: 'application-x-executable',
    run: 'application-x-executable',
    lumen: 'security-high',
  };

  const KIND_ICONS = {
    folder: 'folder',
    image: 'image-x-generic',
    video: 'video-x-generic',
    audio: 'audio-x-generic',
    doc: 'x-office-document',
    text: 'text-x-generic',
    code: 'text-x-generic',
    archive: 'package-x-generic',
    other: 'application-x-executable',
  };

  const RUN_EXTS = new Set([
    'py', 'pyw', 'c', 'cpp', 'cc', 'cxx', 'go', 'rs', 'java', 'kt', 'js', 'mjs', 'cjs', 'ts',
    'sh', 'bash', 'zsh', 'fish', 'rb', 'php', 'pl', 'lua', 'r', 'swift', 'hs', 'ex', 'exs',
    'dart', 'jl', 's', 'asm', 'cs', 'coffee',
  ]);

  const folderIcon = (name) => {
    const lower = name.toLowerCase();
    if (/^(desktop|dolphin|xfce4-desktop)$/.test(lower)) return 'user-desktop';
    if (/^(downloads?|téléchargements)$/.test(lower)) return 'folder-download';
    if (/^(pictures?|photos?|images?|wallpapers?)$/.test(lower)) return 'folder-pictures';
    if (/^(music|audio|sounds?)$/.test(lower)) return 'folder-music';
    if (/^(videos?|movies?)$/.test(lower)) return 'folder-videos';
    if (/^(documents?|docs)$/.test(lower)) return 'folder-documents';
    if (/^(templates?)$/.test(lower)) return 'folder-templates';
    if (/^(public|shared|publicshare)$/.test(lower)) return 'folder-publicshare';
    return 'folder';
  };

  function iconName(e) {
    if (!e) return 'application-x-executable';
    if (e.kind === 'folder') return folderIcon(e.name || '');
    const ext = String(e.ext || '').toLowerCase();
    if (EXT_ICONS[ext]) return EXT_ICONS[ext];
    return KIND_ICONS[e.kind] || 'application-x-executable';
  }

  const fallbackName = (e) => (e && e.kind === 'folder' ? 'folder' : (KIND_ICONS[e && e.kind] || 'application-x-executable'));

  const esc = (s) => String(s).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]);

  function themeURL(name, size) {
    return '/api/icon?' + new URLSearchParams({ name, size: String(Math.round(size)) });
  }

  function themed(name, size, fallback, cls = '') {
    if (!name) return '';
    const fb = api.icon ? api.icon(fallback || 'application-x-executable', size) : '';
    return `<span class="xicon ${cls}" style="width:${size}px;height:${size}px" aria-hidden="true">${fb ? `<span class="xicon-fb">${fb}</span>` : ''}<img class="xicon-img" data-src="${esc(themeURL(name, size))}" alt="" decoding="async" draggable="false"></span>`;
  }

  function placeholder(e, size = 34) {
    if (!bound) return '';
    const ext = size > 30 && e.ext ? `<span class="xext">${esc(e.ext.toUpperCase())}</span>` : '';
    return `<div class="ph xph">${themed(iconName(e), size, fallbackName(e))}${ext}</div>`;
  }

  function appIcon(name, size = 20) {
    return themed(name || 'application-x-executable', size, 'application-x-executable', 'ow-icon');
  }

  function canRun(e) {
    return !!e && e.kind !== 'folder' && RUN_EXTS.has(String(e.ext || '').toLowerCase());
  }

  function bind(obj) {
    Object.assign(api, obj);
    bound = true;
  }

  async function encryptDialog(items) {
    if (!bound || !items.length) return;
    const data = await api.formDialog({
      title: items.length === 1 ? `Encrypt ${items[0].name}` : `Encrypt ${items.length} items`,
      confirm: 'Encrypt',
      body:
        `<label>Password<input type="password" name="pw" autocomplete="new-password" spellcheck="false"></label>` +
        `<label>Repeat the password<input type="password" name="pw2" autocomplete="new-password" spellcheck="false"></label>` +
        `<label class="check"><input type="checkbox" name="trash" value="1"> Move the originals to the trash after encrypting</label>` +
        `<div class="note">The encrypted files get the .lumen extension. Keep the password somewhere safe: it cannot be recovered.</div>`,
    });
    if (!data) return;
    if (String(data.pw || '').length < 4) return api.toast('Choose a password with at least 4 characters', 'error');
    if (data.pw !== data.pw2) return api.toast('The passwords do not match', 'error');
    const done = [];
    const failed = [];
    for (const item of items) {
      try {
        const res = await api.sendJSON('/api/crypt', { action: 'encrypt', path: item.path, password: data.pw, trash: !!data.trash });
        done.push(res.to);
        if (res.warning) api.toast(res.warning, 'error');
      } catch (err) {
        failed.push(`${item.name}: ${err.message}`);
      }
    }
    if (done.length) {
      api.state.pendingSelect = done[0];
      await api.afterOp();
      api.toast(done.length === 1 ? 'Encrypted ' + (items[0].name || 'the file') : `Encrypted ${done.length} items`);
    }
    if (failed.length) api.toast(failed.length === 1 ? failed[0] : `${failed.length} items could not be encrypted: ${failed[0]}`, 'error');
  }

  async function decryptDialog(items) {
    if (!bound || !items.length) return;
    const data = await api.formDialog({
      title: items.length === 1 ? `Decrypt ${items[0].name}` : `Decrypt ${items.length} files`,
      confirm: 'Decrypt',
      body:
        `<label>Password<input type="password" name="pw" autocomplete="current-password" spellcheck="false"></label>` +
        `<div class="note">Folders are restored from their encrypted archive in the same folder.</div>`,
    });
    if (!data) return;
    if (!data.pw) return api.toast('Enter the password', 'error');
    const done = [];
    const failed = [];
    for (const item of items) {
      try {
        const res = await api.sendJSON('/api/crypt', { action: 'decrypt', path: item.path, password: data.pw });
        done.push(res.to);
      } catch (err) {
        failed.push(`${item.name}: ${err.message}`);
      }
    }
    if (done.length) {
      api.state.pendingSelect = done[0];
      await api.afterOp();
      api.toast(done.length === 1 ? 'Decrypted' : `Decrypted ${done.length} files`);
    }
    if (failed.length) api.toast(failed.length === 1 ? failed[0] : `${failed.length} files could not be decrypted: ${failed[0]}`, 'error');
  }

  async function runDialog(e) {
    if (!bound) return;
    let info = {};
    try {
      info = await api.getJSON('/api/dev?' + new URLSearchParams({ path: e.path }));
    } catch {}
    api.dlg.innerHTML =
      `<form method="dialog" class="dlg wide dev-dlg"><h3>Compile &amp; run</h3>` +
      `<p>${esc(e.name)}${info.name ? ` · ${esc(info.name)}` : ''}</p>` +
      `<div class="note">Code runs on this computer with your user permissions. Only run files you trust.</div>` +
      `<label>Arguments (optional)<input type="text" id="dev-args" name="args" autocomplete="off" spellcheck="false" placeholder="--help or a file name"></label>` +
      `<pre class="dev-out" id="dev-out">Ready.</pre>` +
      `<div class="dev-status" id="dev-status"></div>` +
      `<div class="dlg-actions"><button type="button" class="btn" data-close>Close</button><button type="button" class="btn primary" id="dev-run">Run</button></div></form>`;
    api.dlg.returnValue = '';
    api.dlg.showModal();
    const out = api.dlg.querySelector('#dev-out');
    const status = api.dlg.querySelector('#dev-status');
    const args = api.dlg.querySelector('#dev-args');
    const button = api.dlg.querySelector('#dev-run');
    const go = async () => {
      button.disabled = true;
      button.textContent = 'Running…';
      out.classList.remove('err', 'ok');
      out.textContent = 'Running…';
      status.textContent = '';
      status.className = 'dev-status';
      try {
        const res = await api.sendJSON('/api/dev', { path: e.path, args: args.value });
        const lines = [];
        if (res.build) lines.push('Build: ' + res.build);
        if (res.command) lines.push('Run: ' + res.command);
        if (lines.length) lines.push('');
        lines.push(res.output || '(no output)');
        out.textContent = lines.join('\n');
        out.classList.add(res.ok ? 'ok' : 'err');
        status.textContent = res.ok
          ? `Finished in ${(res.durationMs / 1000).toFixed(2)} s`
          : res.timedOut
            ? 'Stopped after the time limit'
            : `Exited with code ${res.exit} after ${(res.durationMs / 1000).toFixed(2)} s`;
        status.className = 'dev-status ' + (res.ok ? 'ok' : 'fail');
      } catch (err) {
        out.textContent = err.message;
        out.classList.add('err');
        status.textContent = 'Could not run this file';
        status.className = 'dev-status fail';
      } finally {
        button.disabled = false;
        button.textContent = 'Run again';
      }
    };
    button.addEventListener('click', go);
    args.addEventListener('keydown', (ev) => {
      if (ev.key === 'Enter') {
        ev.preventDefault();
        go();
      }
    });
    args.focus();
  }

  function itemMenu(sel, readOnly) {
    const items = [];
    if (!bound || !sel.length) return items;
    const encrypted = sel.filter((e) => String(e.ext || '').toLowerCase() === 'lumen');
    if (encrypted.length === sel.length) {
      items.push({ label: sel.length > 1 ? `Decrypt ${sel.length} files…` : 'Decrypt…', disabled: readOnly, run: () => decryptDialog(sel) });
    } else if (!encrypted.length) {
      items.push({ label: sel.length > 1 ? `Encrypt ${sel.length} items…` : 'Encrypt…', disabled: readOnly, run: () => encryptDialog(sel) });
    }
    if (sel.length === 1 && canRun(sel[0])) {
      items.push({ label: 'Compile & run…', run: () => runDialog(sel[0]) });
    }
    return items;
  }

  function openEntry(e) {
    if (!bound || !e || String(e.ext || '').toLowerCase() !== 'lumen') return false;
    decryptDialog([e]);
    return true;
  }

  document.addEventListener(
    'load',
    (ev) => {
      const img = ev.target;
      if (img && img.classList && img.classList.contains('xicon-img')) img.parentElement?.classList.add('has-img');
    },
    true
  );
  document.addEventListener(
    'error',
    (ev) => {
      const img = ev.target;
      if (img && img.classList && img.classList.contains('xicon-img')) img.classList.add('bad');
    },
    true
  );

  if (document.body) {
    watchIcons(document.body);
    new MutationObserver((records) => {
      for (const record of records) {
        for (const node of record.addedNodes) {
          if (node.nodeType === 1) watchIcons(node);
        }
        for (const node of record.removedNodes) {
          if (node.nodeType !== 1) continue;
          const imgs = node.matches?.('.xicon-img[data-src]') ? [node] : [...(node.querySelectorAll?.('.xicon-img[data-src]') || [])];
          for (const img of imgs) if (lazy) lazy.unobserve(img);
        }
      }
    }).observe(document.body, { childList: true, subtree: true });
  }

  window.LumenExtras = { bind, placeholder, appIcon, iconName, canRun, itemMenu, openEntry, runDialog, encryptDialog, decryptDialog, theme: savedTheme, applyTheme, saveTheme, themeOptions };
})();
LUMEN_FILE_6

mkdir -p 'web/css'
cat > 'web/css/extras.css' <<'LUMEN_FILE_7'
.xicon {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  vertical-align: middle;
}
.xicon-fb {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  color: #8f9298;
  pointer-events: none;
}
.xicon-img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: contain;
  pointer-events: none;
}
.xicon-img.bad {
  display: none;
}
.xicon.has-img .xicon-fb {
  display: none;
}
.thumb .xicon-img,
.fprev .xicon-img {
  opacity: 1;
  object-fit: contain;
}
.ph.xph {
  gap: 5px;
}
.ph.xph .xext {
  color: #777b82;
  font-size: 10.5px;
  letter-spacing: 0.08em;
}
.k-image .ph .xext,
.k-video .ph .xext {
  display: none;
}
.dlg input[type="password"] {
  width: 100%;
  background: #1c1d20;
  border: 1px solid #3a3d42;
  border-radius: 7px;
  padding: 8px 10px;
  outline: 0;
}
.dlg input[type="password"]:focus {
  border-color: var(--accent-line);
}
.dlg label.radio .xicon {
  margin-right: 2px;
}
.ow-icon {
  flex: none;
}
.dev-dlg {
  min-width: min(620px, 92vw);
}
.dev-out {
  margin: 0;
  max-height: 320px;
  overflow: auto;
  padding: 10px 12px;
  border: 1px solid #3a3d42;
  border-radius: 8px;
  background: #1a1b1d;
  color: #cfd1d4;
  font: 12px/1.55 "JetBrains Mono", "Fira Code", ui-monospace, monospace;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.dev-out.ok {
  border-color: #3f5a48;
}
.dev-out.err {
  border-color: #7a4b4b;
  color: #ecc3c3;
}
.dev-status {
  min-height: 18px;
  color: var(--dim);
  font-size: 12px;
}
.dev-status.ok {
  color: #9fc6a8;
}
.dev-status.fail {
  color: #cf8f8f;
}
LUMEN_FILE_7

python3 - <<'LUMEN_PATCH_0'
from pathlib import Path
p = Path('internal/server/server.go')
s = p.read_text(encoding='utf-8')
def rep(old, new):
    global s
    have = s.count(old)
    if have == 1:
        s = s.replace(old, new, 1)
        return
    if have == 0 and s.count(new):
        return
    raise SystemExit(f'{p}: expected text not found exactly once')
rep('\t"lumen/internal/config"\n\t"lumen/internal/fsutil"\n\t"lumen/internal/media"\n\t"lumen/internal/search"\n', '\t"lumen/internal/config"\n\t"lumen/internal/fsutil"\n\ticontheme "lumen/internal/icons"\n\t"lumen/internal/media"\n\t"lumen/internal/search"\n')
rep('\thosts    map[string]bool\n\tpreviews previewCache\n}\n\n', '\thosts    map[string]bool\n\tpreviews previewCache\n\ticons    *icontheme.Resolver\n}\n\n')
rep('\t\thosts:    map[string]bool{},\n\t\tpreviews: previewCache{items: map[string]previewEntry{}},\n\t}\n\ts.lastSeen.Store(time.Now().UnixNano())\n', '\t\thosts:    map[string]bool{},\n\t\tpreviews: previewCache{items: map[string]previewEntry{}},\n\t\ticons:    icontheme.New(),\n\t}\n\ts.lastSeen.Store(time.Now().UnixNano())\n')
rep('\tm.HandleFunc("GET /api/openwith", s.openWithList)\n\tm.HandleFunc("POST /api/openwith", s.openWithRun)\n\tm.HandleFunc("POST /api/terminal", s.terminal)\n\tm.HandleFunc("GET /api/sniff", s.sniff)\n', '\tm.HandleFunc("GET /api/openwith", s.openWithList)\n\tm.HandleFunc("POST /api/openwith", s.openWithRun)\n\tm.HandleFunc("GET /api/icon", s.themeIcon)\n\tm.HandleFunc("GET /api/themeicon", s.themeIcon)\n\tm.HandleFunc("GET /api/dev", s.devInfo)\n\tm.HandleFunc("POST /api/dev", s.devRun)\n\tm.HandleFunc("POST /api/crypt", s.crypt)\n\tm.HandleFunc("POST /api/terminal", s.terminal)\n\tm.HandleFunc("GET /api/sniff", s.sniff)\n')
rep('\tcase errors.Is(err, fsutil.ErrExists):\n\t\treturn http.StatusConflict\n\tcase errors.Is(err, fsutil.ErrProtected), errors.Is(err, fsutil.ErrInside):\n\t\treturn http.StatusBadRequest\n', '\tcase errors.Is(err, fsutil.ErrExists):\n\t\treturn http.StatusConflict\n\tcase errors.Is(err, fsutil.ErrPassword):\n\t\treturn http.StatusBadRequest\n\tcase errors.Is(err, fsutil.ErrProtected), errors.Is(err, fsutil.ErrInside):\n\t\treturn http.StatusBadRequest\n')
p.write_text(s, encoding='utf-8')
LUMEN_PATCH_0

python3 - <<'LUMEN_PATCH_1'
from pathlib import Path
p = Path('internal/server/handlers_tools.go')
s = p.read_text(encoding='utf-8')
def rep(old, new):
    global s
    have = s.count(old)
    if have == 1:
        s = s.replace(old, new, 1)
        return
    if have == 0 and s.count(new):
        return
    raise SystemExit(f'{p}: expected text not found exactly once')
rep('\n\t"lumen/internal/fsutil"\n)\n\n', '\n\t"lumen/internal/fsutil"\n\ticontheme "lumen/internal/icons"\n)\n\n')
rep('\tName     string\n\tExec     string\n\tTerminal bool\n\tMimes    map[string]bool\n', '\tName     string\n\tExec     string\n\tIcon     string\n\tTerminal bool\n\tMimes    map[string]bool\n')
rep('\t\tcase "Exec":\n\t\t\tapp.Exec = value\n\t\tcase "Type":\n\t\t\tisApp = value == "Application"\n', '\t\tcase "Exec":\n\t\t\tapp.Exec = value\n\t\tcase "Icon":\n\t\t\tapp.Icon = value\n\t\tcase "Type":\n\t\t\tisApp = value == "Application"\n')
rep('\tID   string `json:"id"`\n\tName string `json:"name"`\n}\n\n', '\tID   string `json:"id"`\n\tName string `json:"name"`\n\tIcon string `json:"icon"`\n}\n\n')
rep('\trecommended, others := []appRow{}, []appRow{}\n\tfor _, a := range loadApps() {\n\t\trow := appRow{ID: a.ID, Name: a.Name}\n\t\tmatch := a.Mimes[mt] || (strings.HasPrefix(mt, "text/") && a.Mimes["text/plain"])\n\t\tswitch {\n', '\trecommended, others := []appRow{}, []appRow{}\n\tfor _, a := range loadApps() {\n\t\ticon := a.Icon\n\t\tif !icontheme.ValidName(icon) {\n\t\t\ticon = "application-x-executable"\n\t\t}\n\t\trow := appRow{ID: a.ID, Name: a.Name, Icon: icon}\n\t\tmatch := a.Mimes[mt] || (strings.HasPrefix(mt, "text/") && a.Mimes["text/plain"])\n\t\tswitch {\n')
p.write_text(s, encoding='utf-8')
LUMEN_PATCH_1

python3 - <<'LUMEN_PATCH_2'
from pathlib import Path
p = Path('web/index.html')
s = p.read_text(encoding='utf-8')
def rep(old, new):
    global s
    have = s.count(old)
    if have == 1:
        s = s.replace(old, new, 1)
        return
    if have == 0 and s.count(new):
        return
    raise SystemExit(f'{p}: expected text not found exactly once')
rep('<link rel="icon" type="image/png" sizes="256x256" href="/icon.png">\n<link rel="stylesheet" href="/css/app.css">\n</head>\n<body>\n', '<link rel="icon" type="image/png" sizes="256x256" href="/icon.png">\n<link rel="stylesheet" href="/css/app.css">\n<link rel="stylesheet" href="/css/extras.css">\n</head>\n<body>\n')
rep('        <option value="audio">Audio</option>\n        <option value="docs">Documents</option>\n        <option value="folders">Folders</option>\n      </select>\n', '        <option value="audio">Audio</option>\n        <option value="docs">Documents</option>\n        <option value="code">Code</option>\n        <option value="text">Text files</option>\n        <option value="archives">Archives</option>\n        <option value="folders">Folders</option>\n      </select>\n')
rep('      <button class="btn small" id="btn-moveto" hidden title="Move the selection to the other pane (F6)">Move</button>\n      <button class="ib" id="btn-newfolder" title="New folder (Ctrl+Shift+N)"></button>\n    </div>\n    <section id="content" tabindex="-1"></section>\n', '      <button class="btn small" id="btn-moveto" hidden title="Move the selection to the other pane (F6)">Move</button>\n      <button class="ib" id="btn-newfolder" title="New folder (Ctrl+Shift+N)"></button>\n      <button class="ib" id="btn-newfile" title="New file"></button>\n    </div>\n    <section id="content" tabindex="-1"></section>\n')
rep('<div id="toasts"></div>\n<script src="/js/highlight.js"></script>\n<script src="/js/app.js"></script>\n</body>\n', '<div id="toasts"></div>\n<script src="/js/highlight.js"></script>\n<script src="/js/extras.js"></script>\n<script src="/js/app.js"></script>\n</body>\n')
p.write_text(s, encoding='utf-8')
LUMEN_PATCH_2

python3 - <<'LUMEN_PATCH_3'
from pathlib import Path
p = Path('web/js/app.js')
s = p.read_text(encoding='utf-8')
def rep(old, new):
    global s
    have = s.count(old)
    if have == 1:
        s = s.replace(old, new, 1)
        return
    if have == 0 and s.count(new):
        return
    raise SystemExit(f'{p}: expected text not found exactly once')
rep('    folder: \'<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>\',\n    newfolder: \'<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><path d="M12 10v5M9.5 12.5h5"/>\',\n    image: \'<rect x="3" y="4" width="18" height="16" rx="2"/><circle cx="9" cy="10" r="1.6"/><path d="m21 16-5-5-8 8"/>\',\n    video: \'<rect x="3" y="5" width="18" height="14" rx="2"/><path d="m10 9 5 3-5 3z"/>\',\n', '    folder: \'<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>\',\n    newfolder: \'<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><path d="M12 10v5M9.5 12.5h5"/>\',\n    newfile: \'<path d="M6 3h8l4 4v14H6z"/><path d="M14 3v4h4M12 10v6M9 13h6"/>\',\n    image: \'<rect x="3" y="4" width="18" height="16" rx="2"/><circle cx="9" cy="10" r="1.6"/><path d="m21 16-5-5-8 8"/>\',\n    video: \'<rect x="3" y="5" width="18" height="14" rx="2"/><path d="m10 9 5 3-5 3z"/>\',\n')
rep("    audio: ['audio'],\n    docs: ['doc', 'text', 'code'],\n    folders: [],\n  };\n", "    audio: ['audio'],\n    docs: ['doc', 'text', 'code'],\n    code: ['code'],\n    text: ['text'],\n    archives: ['archive'],\n    folders: [],\n  };\n")
rep('\n  function placeholder(e, size = 34) {\n    return `<div class="ph">${icon(e.kind === \'other\' || e.kind === \'archive\' || e.kind === \'doc\' ? e.kind : e.kind, size)}${size > 30 && e.ext ? `<span>${esc(e.ext.toUpperCase())}</span>` : \'\'}</div>`;\n  }\n', '\n  function placeholder(e, size = 34) {\n    const themed = window.LumenExtras && LumenExtras.placeholder(e, size);\n    if (themed) return themed;\n    return `<div class="ph">${icon(e.kind === \'other\' || e.kind === \'archive\' || e.kind === \'doc\' ? e.kind : e.kind, size)}${size > 30 && e.ext ? `<span>${esc(e.ext.toUpperCase())}</span>` : \'\'}</div>`;\n  }\n')
rep("    $('#btn-up').disabled = !state.parent || state.mode === 'trash';\n    $('#btn-newfolder').disabled = state.mode !== 'dir' || !state.writable;\n    $('#btn-split').classList.toggle('on', split || embedded);\n    const sendable = (split || embedded) && !!other.path && state.mode !== 'trash';\n", "    $('#btn-up').disabled = !state.parent || state.mode === 'trash';\n    $('#btn-newfolder').disabled = state.mode !== 'dir' || !state.writable;\n    $('#btn-newfile').disabled = state.mode !== 'dir' || !state.writable;\n    $('#btn-split').classList.toggle('on', split || embedded);\n    const sendable = (split || embedded) && !!other.path && state.mode !== 'trash';\n")
rep("\n  function openEntry(e) {\n    if (e.kind === 'folder') navigate(e.path);\n    else if (canView(e)) openViewer(e);\n", "\n  function openEntry(e) {\n    if (window.LumenExtras && LumenExtras.openEntry(e)) return;\n    if (e.kind === 'folder') navigate(e.path);\n    else if (canView(e)) openViewer(e);\n")
rep("      );\n      dlg.showModal();\n      const first = $('input[type=text]', dlg);\n      if (first) {\n        first.focus();\n", "      );\n      dlg.showModal();\n      const first = $('input[type=text], input[type=password]', dlg);\n      if (first) {\n        first.focus();\n")
rep('      return toast(err.message, \'error\');\n    }\n    const row = (a, checked) => `<label class="radio"><input type="radio" name="app" value="${esc(a.id)}" ${checked ? \'checked\' : \'\'}> ${esc(a.name)}${a.id === info.default ? \' (default)\' : \'\'}</label>`;\n    const rec = info.recommended.map((a, i) => row(a, i === 0)).join(\'\');\n    const rest = info.others.map((a) => row(a, false)).join(\'\');\n', '      return toast(err.message, \'error\');\n    }\n    const row = (a, checked) => `<label class="radio"><input type="radio" name="app" value="${esc(a.id)}" ${checked ? \'checked\' : \'\'}> ${window.LumenExtras ? LumenExtras.appIcon(a.icon, 20) : \'\'}<span>${esc(a.name)}${a.id === info.default ? \' (default)\' : \'\'}</span></label>`;\n    const rec = info.recommended.map((a, i) => row(a, i === 0)).join(\'\');\n    const rest = info.others.map((a) => row(a, false)).join(\'\');\n')
rep("      if (one && one.kind !== 'folder') m.push({ label: 'Checksums…', run: () => showChecksums(one) });\n      m.push({ label: sel.length > 1 ? 'Copy paths' : 'Copy path', run: () => copyText(sel.map((x) => x.path).join('\\n')) });\n    }\n    m.push('-', { label: 'Move to trash', key: 'Del', danger: true, disabled: ro, run: trashSelected });\n", "      if (one && one.kind !== 'folder') m.push({ label: 'Checksums…', run: () => showChecksums(one) });\n      m.push({ label: sel.length > 1 ? 'Copy paths' : 'Copy path', run: () => copyText(sel.map((x) => x.path).join('\\n')) });\n      if (window.LumenExtras) {\n        const extra = LumenExtras.itemMenu(sel, ro);\n        if (extra.length) m.push('-', ...extra);\n      }\n    }\n    m.push('-', { label: 'Move to trash', key: 'Del', danger: true, disabled: ro, run: trashSelected });\n")
rep("  }\n  $('#btn-newfolder').addEventListener('click', newFolder);\n  $('#btn-back').addEventListener('click', () => history.back());\n  $('#btn-forward').addEventListener('click', () => history.forward());\n", "  }\n  $('#btn-newfolder').addEventListener('click', newFolder);\n  $('#btn-newfile').addEventListener('click', newFile);\n  $('#btn-back').addEventListener('click', () => history.back());\n  $('#btn-forward').addEventListener('click', () => history.forward());\n")
rep("    let ta = null;\n\n    const viewButtons = () => setButtons([['wrap', 'Wrap'], ['copy', 'Copy'], ...(canEdit ? [['edit', 'Edit', true]] : [])]);\n    const setDirty = (on) => {\n      viewer.dirty = on;\n", "    let ta = null;\n\n    const viewButtons = () => setButtons([['wrap', 'Wrap'], ['copy', 'Copy'], ...(window.LumenExtras && LumenExtras.canRun(e) ? [['run', 'Run']] : []), ...(canEdit ? [['edit', 'Edit', true]] : [])]);\n    const setDirty = (on) => {\n      viewer.dirty = on;\n")
rep("      else if (act === 'wrap') store.set('wrap', body.classList.toggle('wrap'));\n      else if (act === 'edit') renderEdit();\n      else if (act === 'save') saveText();\n      else if (act === 'done') leaveEdit();\n", "      else if (act === 'wrap') store.set('wrap', body.classList.toggle('wrap'));\n      else if (act === 'edit') renderEdit();\n      else if (act === 'run') LumenExtras.runDialog(e);\n      else if (act === 'save') saveText();\n      else if (act === 'done') leaveEdit();\n")
rep("    $('#sfilter').innerHTML = icon('filter', 16);\n    $('#btn-newfolder').innerHTML = icon('newfolder');\n    $('#btn-newtab').innerHTML = icon('plus');\n    $('#btn-split').innerHTML = icon('split');\n", "    $('#sfilter').innerHTML = icon('filter', 16);\n    $('#btn-newfolder').innerHTML = icon('newfolder');\n    $('#btn-newfile').innerHTML = icon('newfile');\n    $('#btn-newtab').innerHTML = icon('plus');\n    $('#btn-split').innerHTML = icon('split');\n")
rep('  }\n\n  boot();\n})();\n', '  }\n\n  if (window.LumenExtras) {\n    LumenExtras.bind({ icon, formDialog, toast, sendJSON, getJSON, state, afterOp, dlg });\n  }\n  boot();\n})();\n')
p.write_text(s, encoding='utf-8')
LUMEN_PATCH_3

python3 - <<'LUMEN_PATCH_4'
from pathlib import Path
p = Path('main.go')
s = p.read_text(encoding='utf-8')
def rep(old, new):
    global s
    have = s.count(old)
    if have == 1:
        s = s.replace(old, new, 1)
        return
    if have == 0 and s.count(new):
        return
    raise SystemExit(f'{p}: expected text not found exactly once')
rep('\tscript := filepath.Join(dataHome, "lumen", "window.py")\n\tif _, err := os.Stat(script); err != nil {\n\t\treturn nil\n\t}\n', '\tscript := filepath.Join(dataHome, "lumen", "window.py")\n\tif _, err := os.Stat(script); err != nil {\n\t\texe, err := os.Executable()\n\t\tif err != nil {\n\t\t\treturn nil\n\t\t}\n\t\tscript = filepath.Join(filepath.Dir(exe), "packaging", "window.py")\n\t\tif _, err := os.Stat(script); err != nil {\n\t\t\treturn nil\n\t\t}\n\t}\n')
p.write_text(s, encoding='utf-8')
p = Path('README.md')
s = p.read_text(encoding='utf-8')
rep('Run: `lumen [folder or file]` opens a standalone window (needs a Chromium-based browser). Options:', 'Run: `lumen [folder or file]` opens as a standalone desktop window. Options:')
p.write_text(s, encoding='utf-8')
LUMEN_PATCH_4

python3 - <<'LUMEN_PATCH_5'
from pathlib import Path
p = Path('web/css/app.css')
s = p.read_text(encoding='utf-8')
def rep(old, new):
    global s
    have = s.count(old)
    if have == 1:
        s = s.replace(old, new, 1)
        return
    if have == 0 and s.count(new):
        return
    raise SystemExit(f'{p}: expected text not found exactly once')
rep(""":root {
  --bg: #1a1b1d;
  --panel: #202124;
  --raised: #27282c;
  --hover: #2d2f33;
  --line: #313337;
  --text: #d8d9dc;
  --dim: #8c8f96;
  --faint: #5f6268;
  --accent: #86a9ad;
  --accent-soft: rgba(134, 169, 173, 0.16);
  --accent-ink: #14201f;
  --danger: #cf8f8f;
  --radius: 8px;
  color-scheme: dark;
}
""", """:root {
  --bg: #1b191c;
  --panel: #211f22;
  --raised: #28252a;
  --hover: #2e2b30;
  --line: #332f35;
  --text: #dad8db;
  --dim: #918c93;
  --faint: #655f67;
  --accent: #c49aa7;
  --accent-soft: rgba(196, 154, 167, 0.15);
  --accent-line: #a57786;
  --accent-glow: rgba(196, 154, 167, 0.3);
  --accent-ink: #241a1f;
  --danger: #cf8f8f;
  --radius: 8px;
  color-scheme: dark;
}

html[data-theme="rose"] {
  --bg: #1f171c;
  --panel: #261c22;
  --raised: #2e2229;
  --hover: #362830;
  --line: #42313a;
  --text: #e0d7db;
  --dim: #9d8c95;
  --faint: #705f68;
  --accent: #dd8faa;
  --accent-soft: rgba(221, 143, 170, 0.18);
  --accent-line: #b96e8a;
  --accent-glow: rgba(221, 143, 170, 0.36);
  --accent-ink: #2a1620;
}

html[data-theme="red"] {
  --bg: #1c1919;
  --panel: #221f1f;
  --raised: #2a2626;
  --hover: #302c2c;
  --line: #373131;
  --text: #dcd8d8;
  --dim: #938d8d;
  --faint: #686060;
  --accent: #bf9696;
  --accent-soft: rgba(191, 150, 150, 0.15);
  --accent-line: #9e6d6d;
  --accent-glow: rgba(191, 150, 150, 0.3);
  --accent-ink: #241919;
}

html[data-theme="blue"] {
  --bg: #181a1d;
  --panel: #1e2124;
  --raised: #25292d;
  --hover: #2b3035;
  --line: #31373c;
  --text: #d7dade;
  --dim: #8d939a;
  --faint: #616970;
  --accent: #9db2c3;
  --accent-soft: rgba(157, 178, 195, 0.15);
  --accent-line: #728ca0;
  --accent-glow: rgba(157, 178, 195, 0.3);
  --accent-ink: #172027;
}

html[data-theme="green"] {
  --bg: #191b19;
  --panel: #1f221f;
  --raised: #262a26;
  --hover: #2c302c;
  --line: #333833;
  --text: #d9dcd8;
  --dim: #8f948d;
  --faint: #636860;
  --accent: #a3b9a7;
  --accent-soft: rgba(163, 185, 167, 0.15);
  --accent-line: #78927c;
  --accent-glow: rgba(163, 185, 167, 0.3);
  --accent-ink: #19231b;
}

html[data-theme="gray"] {
  --bg: #1a1b1d;
  --panel: #202124;
  --raised: #27282c;
  --hover: #2d2f33;
  --line: #313337;
  --text: #d8d9dc;
  --dim: #8c8f96;
  --faint: #5f6268;
  --accent: #aaadb2;
  --accent-soft: rgba(170, 173, 178, 0.14);
  --accent-line: #83878d;
  --accent-glow: rgba(170, 173, 178, 0.28);
  --accent-ink: #1b1d20;
}
""")
rep('.mark { width: 18px; height: 18px; border-radius: 50%; border: 2.5px solid var(--accent); box-shadow: 0 0 12px rgba(134, 169, 173, 0.35); }', '.mark { width: 18px; height: 18px; border-radius: 50%; border: 2.5px solid var(--accent); box-shadow: 0 0 12px var(--accent-glow); }')
rep('.pl.active { background: var(--accent-soft); color: #e3eeee; }', '.pl.active { background: var(--accent-soft); color: var(--text); }')
rep('.ib.on { background: var(--accent-soft); color: #dfeceb; }', '.ib.on { background: var(--accent-soft); color: var(--text); }')
rep('.searchbox:focus-within { border-color: #46797f; }', '.searchbox:focus-within { border-color: var(--accent-line); }')
rep('#scope.wide { background: var(--accent-soft); color: #cfe3e2; }', '#scope.wide { background: var(--accent-soft); color: var(--text); }')
rep('.fbody .ic { color: #9bb5b8; }', '.fbody .ic { color: var(--accent); }')
rep('.item.sel .thumb::after { content: ""; position: absolute; inset: 0; background: rgba(134, 169, 173, 0.18); }', '.item.sel .thumb::after { content: ""; position: absolute; inset: 0; background: var(--accent-soft); }')
rep('.btn.primary { background: #5f8a8f; color: #0f1a1a; font-weight: 500; }\n.btn.primary:hover { background: #6e9a9f; }', '.btn.primary { background: var(--accent); color: var(--accent-ink); font-weight: 500; }\n.btn.primary:hover { background: var(--accent); filter: brightness(1.08); }')
rep('.dlg input[type=text]:focus, .dlg textarea:focus { border-color: #5f8a8f; }', '.dlg input[type=text]:focus, .dlg textarea:focus { border-color: var(--accent-line); }')
rep('.chip.on { background: var(--accent-soft); border-color: #5f8a8f; color: #dcebea; }', '.chip.on { background: var(--accent-soft); border-color: var(--accent-line); color: var(--text); }')
rep('.sp-hint code { background: #1f2023; padding: 1px 6px; border-radius: 5px; color: #b9c7c8; }', '.sp-hint code { background: #1f2023; padding: 1px 6px; border-radius: 5px; color: var(--accent); }')
rep('.dlg code { background: #1f2023; padding: 1px 6px; border-radius: 5px; color: #b9c7c8; }', '.dlg code { background: #1f2023; padding: 1px 6px; border-radius: 5px; color: var(--accent); }')
rep('.cv-lang { padding: 2px 10px; border-radius: 999px; background: rgba(255, 111, 168, 0.16); color: #ff9cc6; font-weight: 600; }', '.cv-lang { padding: 2px 10px; border-radius: 999px; background: var(--accent-soft); color: var(--accent); font-weight: 600; }')
rep('.cv-body ::selection { background: rgba(255, 111, 168, 0.28); }', '.cv-body ::selection { background: var(--accent-soft); }')
rep('.cv-dirty { color: #ff9cc6; font-size: 12px; }\n.cv-btn.primary { background: rgba(255, 111, 168, 0.18); border-color: #7d3a58; color: #ffc2dc; }\n.cv-btn.primary:hover { background: rgba(255, 111, 168, 0.3); }', '.cv-dirty { color: var(--accent); font-size: 12px; }\n.cv-btn.primary { background: var(--accent-soft); border-color: var(--accent-line); color: var(--text); }\n.cv-btn.primary:hover { background: var(--accent); color: var(--accent-ink); }')
rep('.ed-ta { position: absolute; inset: 0; width: 100%; height: 100%; margin: 0; padding: 0 28px 0 0; border: 0; outline: 0; resize: none; overflow: hidden; background: transparent; color: transparent; caret-color: #ff6fa8; font: inherit; white-space: pre; tab-size: 4; box-sizing: border-box; }\n.ed-ta::selection { background: rgba(255, 111, 168, 0.3); color: transparent; }', '.ed-ta { position: absolute; inset: 0; width: 100%; height: 100%; margin: 0; padding: 0 28px 0 0; border: 0; outline: 0; resize: none; overflow: hidden; background: transparent; color: transparent; caret-color: var(--accent); font: inherit; white-space: pre; tab-size: 4; box-sizing: border-box; }\n.ed-ta::selection { background: var(--accent-soft); color: transparent; }')
p.write_text(s, encoding='utf-8')
LUMEN_PATCH_5

python3 - <<'LUMEN_PATCH_6'
from pathlib import Path
p = Path('web/index.html')
s = p.read_text(encoding='utf-8')
def rep(old, new):
    global s
    have = s.count(old)
    if have == 1:
        s = s.replace(old, new, 1)
        return
    if have == 0 and s.count(new):
        return
    raise SystemExit(f'{p}: expected text not found exactly once')
rep('<meta name="color-scheme" content="dark">\n<title>Lumen</title>\n', '<meta name="color-scheme" content="dark">\n<script>try{const t=JSON.parse(localStorage.getItem(\'lumen.theme\')||\'"rose"\');document.documentElement.dataset.theme=/^(rose|red|blue|green|gray)$/.test(t)?t:\'rose\'}catch(e){document.documentElement.dataset.theme=\'rose\'}</script>\n<title>Lumen</title>\n')
p.write_text(s, encoding='utf-8')
LUMEN_PATCH_6

python3 - <<'LUMEN_PATCH_7'
from pathlib import Path
p = Path('web/js/app.js')
s = p.read_text(encoding='utf-8')
def rep(old, new):
    global s
    have = s.count(old)
    if have == 1:
        s = s.replace(old, new, 1)
        return
    if have == 0 and s.count(new):
        return
    raise SystemExit(f'{p}: expected text not found exactly once')
rep('      <label>Folder names to skip while indexing (comma separated)<input type="text" name="skip" value="${esc(cfg.skipDirs.join(\', \'))}" spellcheck="false"></label>\n      <label class="check"><input type="checkbox" name="single" ${state.singleClick ? \'checked\' : \'\'}> Open items with a single click</label>\n', '      <label>Folder names to skip while indexing (comma separated)<input type="text" name="skip" value="${esc(cfg.skipDirs.join(\', \'))}" spellcheck="false"></label>\n      <label>Theme<select name="theme">${window.LumenExtras ? LumenExtras.themeOptions() : \'\'}</select></label>\n      <label class="check"><input type="checkbox" name="single" ${state.singleClick ? \'checked\' : \'\'}> Open items with a single click</label>\n')
rep("    dlg.returnValue = '';\n    $('#s-reindex').onclick = async () => {\n", "    dlg.returnValue = '';\n    const theme = window.LumenExtras ? LumenExtras.theme() : 'rose';\n    const themePick = $('select[name=theme]', dlg);\n    if (themePick && window.LumenExtras) themePick.addEventListener('change', () => LumenExtras.applyTheme(themePick.value));\n    $('#s-reindex').onclick = async () => {\n")
rep("      async () => {\n        if (dlg.returnValue !== 'ok') return;\n        const form = $('form', dlg);\n        const roots = form.roots.value.split('\\n').map((s) => s.trim()).filter(Boolean);\n", "      async () => {\n        if (dlg.returnValue !== 'ok') {\n          if (window.LumenExtras) LumenExtras.applyTheme(theme);\n          return;\n        }\n        if (window.LumenExtras && themePick) LumenExtras.saveTheme(themePick.value);\n        const form = $('form', dlg);\n        const roots = form.roots.value.split('\\n').map((s) => s.trim()).filter(Boolean);\n")
p.write_text(s, encoding='utf-8')
LUMEN_PATCH_7

python3 - <<'LUMEN_PATCH_8'
from pathlib import Path
p = Path('web/css/app.css')
s = p.read_text(encoding='utf-8')
if 'html[data-theme="rose"]' not in s:
    rose = """html[data-theme="rose"] {
  --bg: #1f171c;
  --panel: #261c22;
  --raised: #2e2229;
  --hover: #362830;
  --line: #42313a;
  --text: #e0d7db;
  --dim: #9d8c95;
  --faint: #705f68;
  --accent: #dd8faa;
  --accent-soft: rgba(221, 143, 170, 0.18);
  --accent-line: #b96e8a;
  --accent-glow: rgba(221, 143, 170, 0.36);
  --accent-ink: #2a1620;
}

"""
    anchor = 'html[data-theme="red"] {'
    if s.count(anchor) != 1:
        raise SystemExit(f'{p}: expected text not found exactly once')
    s = s.replace(anchor, rose + anchor, 1)
p.write_text(s, encoding='utf-8')
LUMEN_PATCH_8

gofmt -w internal/dev/dev.go internal/fsutil/crypt.go internal/icons/icons.go internal/server/handlers_crypt.go internal/server/handlers_dev.go internal/server/handlers_icons.go internal/server/handlers_tools.go internal/server/server.go
go vet ./...
go build -o lumen .
if command -v node >/dev/null 2>&1; then
  node --check web/js/extras.js
  node --check web/js/app.js
fi
chmod +x lumen
echo "Upgrade complete. Start Lumen as a desktop app with ./lumen"
echo "For an application-menu icon, run: ./install.sh"
