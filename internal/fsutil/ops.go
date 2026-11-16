package fsutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

var (
	ErrExists    = errors.New("an item with that name already exists")
	ErrProtected = errors.New("this location is protected")
	ErrInside    = errors.New("cannot place a folder inside itself")
)

func ValidName(name string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return errors.New("the name cannot be empty")
	case name == "." || name == "..":
		return errors.New("that name is not allowed")
	case strings.ContainsAny(name, "/\x00"):
		return errors.New("the name cannot contain a slash")
	case len(name) > 255:
		return errors.New("the name is too long")
	}
	return nil
}

func Protected(path string) bool {
	clean := filepath.Clean(path)
	if clean == "/" {
		return true
	}
	if home, err := os.UserHomeDir(); err == nil && clean == filepath.Clean(home) {
		return true
	}
	return false
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func splitName(name string) (string, string) {
	lower := strings.ToLower(name)
	for _, double := range []string{".tar.gz", ".tar.xz", ".tar.bz2", ".tar.zst"} {
		if strings.HasSuffix(lower, double) && len(name) > len(double) {
			return name[:len(name)-len(double)], name[len(name)-len(double):]
		}
	}
	ext := filepath.Ext(name)
	if ext == "" || ext == name {
		return name, ""
	}
	return strings.TrimSuffix(name, ext), ext
}

func UniqueName(dir, name string) string {
	if !exists(filepath.Join(dir, name)) {
		return name
	}
	base, ext := splitName(name)
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", base, i, ext)
		if !exists(filepath.Join(dir, candidate)) {
			return candidate
		}
	}
}

func Rename(path, newName string) (string, error) {
	if Protected(path) {
		return "", ErrProtected
	}
	if err := ValidName(newName); err != nil {
		return "", err
	}
	target := filepath.Join(filepath.Dir(path), newName)
	if target == path {
		return path, nil
	}
	if exists(target) {
		return "", ErrExists
	}
	if err := os.Rename(path, target); err != nil {
		return "", err
	}
	return target, nil
}

func Mkdir(parent, name string) (string, error) {
	if err := ValidName(name); err != nil {
		return "", err
	}
	target := filepath.Join(parent, name)
	if exists(target) {
		return "", ErrExists
	}
	if err := os.Mkdir(target, 0o755); err != nil {
		return "", err
	}
	return target, nil
}

func isInside(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	if parent == child {
		return true
	}
	if parent == "/" {
		return true
	}
	return strings.HasPrefix(child, parent+"/")
}

func Copy(src, dstDir string) (string, error) {
	info, err := os.Lstat(src)
	if err != nil {
		return "", err
	}
	if info.IsDir() && isInside(src, dstDir) {
		return "", ErrInside
	}
	target := filepath.Join(dstDir, UniqueName(dstDir, filepath.Base(src)))
	if err := copyTree(src, target); err != nil {
		os.RemoveAll(target)
		return "", err
	}
	return target, nil
}

func copyTree(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	switch {
	case info.Mode()&fs.ModeSymlink != 0:
		link, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(link, dst)
	case info.IsDir():
		if err := os.Mkdir(dst, info.Mode().Perm()|0o700); err != nil {
			return err
		}
		items, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, item := range items {
			if err := copyTree(filepath.Join(src, item.Name()), filepath.Join(dst, item.Name())); err != nil {
				return err
			}
		}
		return os.Chmod(dst, info.Mode().Perm())
	case info.Mode().IsRegular():
		return copyFile(src, dst, info)
	}
	return nil
}

func copyFile(src, dst string, info fs.FileInfo) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chtimes(dst, info.ModTime(), info.ModTime())
}

func Move(src, dstDir string) (string, error) {
	if Protected(src) {
		return "", ErrProtected
	}
	info, err := os.Lstat(src)
	if err != nil {
		return "", err
	}
	if filepath.Dir(src) == filepath.Clean(dstDir) {
		return src, nil
	}
	if info.IsDir() && isInside(src, dstDir) {
		return "", ErrInside
	}
	target := filepath.Join(dstDir, UniqueName(dstDir, filepath.Base(src)))
	err = os.Rename(src, target)
	if err == nil {
		return target, nil
	}
	if !errors.Is(err, syscall.EXDEV) {
		return "", err
	}
	if err := copyTree(src, target); err != nil {
		os.RemoveAll(target)
		return "", err
	}
	if err := os.RemoveAll(src); err != nil {
		return target, err
	}
	return target, nil
}

func Remove(path string) error {
	if Protected(path) {
		return ErrProtected
	}
	return os.RemoveAll(path)
}

type Stats struct {
	Size    int64 `json:"size"`
	Files   int   `json:"files"`
	Folders int   `json:"folders"`
}

func DirStats(ctx context.Context, root string) (Stats, error) {
	var st Stats
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if path != root {
				st.Folders++
			}
			return nil
		}
		st.Files++
		if info, err := d.Info(); err == nil && info.Mode().IsRegular() {
			st.Size += info.Size()
		}
		return nil
	})
	return st, err
}
