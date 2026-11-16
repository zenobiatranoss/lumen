package fsutil

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var archiveTypes = []struct{ suffix, kind string }{
	{".tar.gz", "tgz"}, {".tgz", "tgz"}, {".tar.bz2", "tbz"}, {".tbz2", "tbz"}, {".tbz", "tbz"},
	{".tar.xz", "txz"}, {".txz", "txz"}, {".tar", "tar"}, {".zip", "zip"}, {".jar", "zip"}, {".apk", "zip"},
	{".7z", "7z"}, {".rar", "rar"}, {".iso", "iso"}, {".gz", "gz"}, {".bz2", "bz2"}, {".xz", "xz"},
}

var sevenZipNames = []string{"7zz", "7z", "7za", "7zr"}

func ArchiveKind(name string) string {
	lower := strings.ToLower(name)
	for _, t := range archiveTypes {
		if strings.HasSuffix(lower, t.suffix) && len(lower) > len(t.suffix) {
			return t.kind
		}
	}
	return ""
}

func archiveBase(name string) string {
	lower := strings.ToLower(name)
	for _, t := range archiveTypes {
		if strings.HasSuffix(lower, t.suffix) && len(name) > len(t.suffix) {
			return name[:len(name)-len(t.suffix)]
		}
	}
	return name
}

var compressExt = map[string]string{"zip": ".zip", "tgz": ".tar.gz", "txz": ".tar.xz", "7z": ".7z"}

func Compress(paths []string, dstDir, name, format string) (string, error) {
	ext, ok := compressExt[format]
	if !ok {
		return "", errors.New("unknown archive format")
	}
	if len(paths) == 0 {
		return "", errors.New("nothing to compress")
	}
	name = strings.TrimSpace(name)
	if err := ValidName(name + ext); err != nil {
		return "", err
	}
	out := filepath.Join(dstDir, UniqueName(dstDir, name+ext))
	part := out + ".part"
	var err error
	switch format {
	case "zip":
		err = writeZip(paths, part)
	case "tgz":
		err = writeTarGz(paths, part)
	case "txz":
		err = runTar(paths, part)
	case "7z":
		err = run7z(paths, part)
	}
	if err != nil {
		os.Remove(part)
		return "", err
	}
	if err := os.Rename(part, out); err != nil {
		os.Remove(part)
		return "", err
	}
	return out, nil
}

func lastLine(data []byte) string {
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

func runTool(bin, dir string, args ...string) error {
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	data, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %s", filepath.Base(bin), lastLine(data))
	}
	return nil
}

func findTool(names ...string) string {
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
}

func runTar(paths []string, out string) error {
	bin := findTool("tar")
	if bin == "" {
		return errors.New("tar is not installed")
	}
	args := []string{"-cJf", out}
	for _, p := range paths {
		args = append(args, "-C", filepath.Dir(p), filepath.Base(p))
	}
	return runTool(bin, "", args...)
}

func run7z(paths []string, out string) error {
	bin := findTool(sevenZipNames...)
	if bin == "" {
		return errors.New("7-Zip is not installed, install it with: sudo apt install 7zip")
	}
	parent := filepath.Dir(paths[0])
	args := []string{"a", "-t7z", "-y", "-bd", out}
	for _, p := range paths {
		if filepath.Dir(p) != parent {
			return errors.New("for 7z all items must be in the same folder")
		}
		args = append(args, filepath.Base(p))
	}
	return runTool(bin, parent, args...)
}

func walkEntries(paths []string, skip string, fn func(path, rel string, info fs.FileInfo, link string) error) error {
	for _, p := range paths {
		parent := filepath.Dir(p)
		err := filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if path == skip {
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
			rel, _ := filepath.Rel(parent, path)
			return fn(path, filepath.ToSlash(rel), info, link)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func copyFileInto(w io.Writer, path string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()
	_, err = io.Copy(w, src)
	return err
}

func writeZip(paths []string, out string) error {
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	err = walkEntries(paths, out, func(path, rel string, info fs.FileInfo, link string) error {
		hdr, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		hdr.Name = rel
		if info.IsDir() {
			hdr.Name += "/"
			hdr.Method = zip.Store
			_, err = zw.CreateHeader(hdr)
			return err
		}
		hdr.Method = zip.Deflate
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}
		if link != "" {
			_, err = io.WriteString(w, link)
			return err
		}
		return copyFileInto(w, path)
	})
	if err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return f.Close()
}

func writeTarGz(paths []string, out string) error {
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	err = walkEntries(paths, out, func(path, rel string, info fs.FileInfo, link string) error {
		hdr, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return err
		}
		hdr.Name = rel
		if info.IsDir() {
			hdr.Name += "/"
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		return copyFileInto(tw, path)
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

func Extract(src, dstDir string) (string, error) {
	kind := ArchiveKind(filepath.Base(src))
	if kind == "" {
		return "", errors.New("this file is not a supported archive")
	}
	base := archiveBase(filepath.Base(src))
	out := filepath.Join(dstDir, UniqueName(dstDir, base))
	if err := os.Mkdir(out, 0o755); err != nil {
		return "", err
	}
	var err error
	switch kind {
	case "zip":
		err = extractZip(src, out)
	case "tar", "tgz", "tbz":
		err = extractTar(src, out, kind)
	case "gz", "bz2", "xz":
		err = extractSingle(src, out, kind)
	default:
		err = extractExternal(src, out, kind)
	}
	if err != nil {
		os.RemoveAll(out)
		return "", err
	}
	return out, nil
}

func safeJoin(root, name string) (string, error) {
	target := filepath.Join(root, filepath.FromSlash(name))
	if target != root && !strings.HasPrefix(target, root+string(os.PathSeparator)) {
		return "", errors.New("the archive contains an unsafe path")
	}
	return target, nil
}

func writeFile(target string, r io.Reader, perm fs.FileMode) error {
	if perm == 0 {
		perm = 0o644
	}
	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm|0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func extractZip(src, out string) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		target, err := safeJoin(out, f.Name)
		if err != nil {
			return err
		}
		mode := f.Mode()
		if mode&os.ModeSymlink != 0 {
			continue
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		err = writeFile(target, rc, mode.Perm())
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTar(src, out, kind string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	var r io.Reader = f
	switch kind {
	case "tgz":
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gz.Close()
		r = gz
	case "tbz":
		r = bzip2.NewReader(f)
	}
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target, err := safeJoin(out, hdr.Name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := writeFile(target, tr, fs.FileMode(hdr.Mode).Perm()); err != nil {
				return err
			}
		}
	}
}

func extractSingle(src, out, kind string) error {
	dst, err := os.Create(filepath.Join(out, archiveBase(filepath.Base(src))))
	if err != nil {
		return err
	}
	defer dst.Close()
	if kind == "xz" {
		cmd := exec.Command("xz", "-dc", src)
		cmd.Stdout = dst
		if err := cmd.Run(); err != nil {
			return errors.New("xz could not unpack this file")
		}
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if kind == "gz" {
		gz, err := gzip.NewReader(in)
		if err != nil {
			return err
		}
		defer gz.Close()
		_, err = io.Copy(dst, gz)
		return err
	}
	_, err = io.Copy(dst, bzip2.NewReader(in))
	return err
}

func extractExternal(src, out, kind string) error {
	var tries [][]string
	sevenZip := func(bin string) []string {
		return []string{bin, "x", "-y", "-bd", "-o" + out, src}
	}
	switch kind {
	case "txz":
		tries = append(tries, []string{"tar", "-xJf", src, "-C", out}, []string{"bsdtar", "-xf", src, "-C", out})
	case "rar":
		tries = append(tries, []string{"unrar", "x", "-y", "-idq", src, out + "/"}, []string{"unar", "-q", "-o", out, src})
		for _, n := range sevenZipNames {
			tries = append(tries, sevenZip(n))
		}
		tries = append(tries, []string{"bsdtar", "-xf", src, "-C", out})
	default:
		for _, n := range sevenZipNames {
			tries = append(tries, sevenZip(n))
		}
		tries = append(tries, []string{"bsdtar", "-xf", src, "-C", out}, []string{"unar", "-q", "-o", out, src})
	}
	last := errors.New("no tool was found to open this archive, install one with: sudo apt install 7zip unrar-free")
	for _, t := range tries {
		bin, err := exec.LookPath(t[0])
		if err != nil {
			continue
		}
		if err := runTool(bin, "", t[1:]...); err != nil {
			last = err
			continue
		}
		return nil
	}
	return last
}

func NewFile(parent, name string) (string, error) {
	if err := ValidName(name); err != nil {
		return "", err
	}
	target := filepath.Join(parent, UniqueName(parent, name))
	f, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	return target, f.Close()
}

func Symlink(target, dir, name string) (string, error) {
	if err := ValidName(name); err != nil {
		return "", err
	}
	link := filepath.Join(dir, UniqueName(dir, name))
	return link, os.Symlink(target, link)
}
