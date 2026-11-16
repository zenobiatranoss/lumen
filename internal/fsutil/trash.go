package fsutil

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

type TrashItem struct {
	Entry
	Original string `json:"original"`
	Deleted  string `json:"deleted"`
}

const infoSuffix = ".trashinfo"

func dataHome() string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share")
}

func homeTrash() string {
	return filepath.Join(dataHome(), "Trash")
}

func topTrash(mount string) string {
	return filepath.Join(mount, fmt.Sprintf(".Trash-%d", os.Getuid()))
}

func trashRoots() []string {
	roots := []string{homeTrash()}
	for _, d := range Drives() {
		if d.Path == "/" {
			continue
		}
		candidate := topTrash(d.Path)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			roots = append(roots, candidate)
		}
	}
	return roots
}

func deviceOf(path string) (uint64, error) {
	var st syscall.Stat_t
	if err := syscall.Lstat(path, &st); err != nil {
		return 0, err
	}
	return uint64(st.Dev), nil
}

func mountPoint(path string) string {
	dev, err := deviceOf(path)
	if err != nil {
		return "/"
	}
	current := path
	for {
		parent := filepath.Dir(current)
		if parent == current {
			return current
		}
		pdev, err := deviceOf(parent)
		if err != nil || pdev != dev {
			return current
		}
		current = parent
	}
}

func prepareTrash(root string) error {
	if err := os.MkdirAll(filepath.Join(root, "files"), 0o700); err != nil {
		return err
	}
	return os.MkdirAll(filepath.Join(root, "info"), 0o700)
}

func encodeTrashPath(p string) string {
	u := url.URL{Path: p}
	return u.EscapedPath()
}

func claimName(root, base string) (string, *os.File, error) {
	name := base
	for i := 2; i < 10000; i++ {
		infoPath := filepath.Join(root, "info", name+infoSuffix)
		file, err := os.OpenFile(infoPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			if !exists(filepath.Join(root, "files", name)) {
				return name, file, nil
			}
			file.Close()
			os.Remove(infoPath)
		} else if !errors.Is(err, os.ErrExist) {
			return "", nil, err
		}
		name = fmt.Sprintf("%s.%d", base, i)
	}
	return "", nil, errors.New("could not find a free name in the trash")
}

func MoveToTrash(path string) error {
	if Protected(path) {
		return ErrProtected
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(abs); err != nil {
		return err
	}
	root := homeTrash()
	stored := abs
	if err := prepareTrash(root); err != nil {
		return err
	}
	fileDev, err := deviceOf(abs)
	if err != nil {
		return err
	}
	trashDev, err := deviceOf(root)
	if err != nil {
		return err
	}
	sameDevice := fileDev == trashDev
	if !sameDevice {
		mount := mountPoint(abs)
		candidate := topTrash(mount)
		if prepareTrash(candidate) == nil {
			root = candidate
			if rel, err := filepath.Rel(mount, abs); err == nil {
				stored = rel
			}
			sameDevice = true
		}
	}
	name, infoFile, err := claimName(root, filepath.Base(abs))
	if err != nil {
		return err
	}
	infoPath := infoFile.Name()
	content := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n", encodeTrashPath(stored), time.Now().Format("2006-01-02T15:04:05"))
	_, werr := infoFile.WriteString(content)
	cerr := infoFile.Close()
	if werr != nil || cerr != nil {
		os.Remove(infoPath)
		return errors.Join(werr, cerr)
	}
	dest := filepath.Join(root, "files", name)
	if sameDevice {
		if err := os.Rename(abs, dest); err != nil {
			os.Remove(infoPath)
			return err
		}
		return nil
	}
	if err := copyTree(abs, dest); err != nil {
		os.RemoveAll(dest)
		os.Remove(infoPath)
		return err
	}
	return os.RemoveAll(abs)
}

func parseInfo(infoPath, topdir string) (string, string) {
	data, err := os.ReadFile(infoPath)
	if err != nil {
		return "", ""
	}
	var original, deleted string
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch key {
		case "Path":
			if decoded, err := url.PathUnescape(value); err == nil {
				original = decoded
			}
		case "DeletionDate":
			deleted = value
		}
	}
	if original != "" && !filepath.IsAbs(original) && topdir != "" {
		original = filepath.Join(topdir, original)
	}
	return original, deleted
}

func ListTrash() ([]TrashItem, error) {
	home := homeTrash()
	var items []TrashItem
	for _, root := range trashRoots() {
		topdir := ""
		if root != home {
			topdir = filepath.Dir(root)
		}
		files, err := os.ReadDir(filepath.Join(root, "files"))
		if err != nil {
			continue
		}
		for _, f := range files {
			full := filepath.Join(root, "files", f.Name())
			info, err := f.Info()
			if err != nil {
				continue
			}
			entry := EntryFromInfo(full, info)
			original, deleted := parseInfo(filepath.Join(root, "info", f.Name()+infoSuffix), topdir)
			items = append(items, TrashItem{Entry: entry, Original: original, Deleted: deleted})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Deleted > items[j].Deleted })
	return items, nil
}

func trashRootOf(path string) (string, error) {
	clean := filepath.Clean(path)
	for _, root := range trashRoots() {
		files := filepath.Join(root, "files")
		if strings.HasPrefix(clean, files+"/") && filepath.Dir(clean) == files {
			return root, nil
		}
	}
	return "", errors.New("that item is not in the trash")
}

func RestoreTrash(path string) (string, error) {
	root, err := trashRootOf(path)
	if err != nil {
		return "", err
	}
	name := filepath.Base(path)
	topdir := ""
	if root != homeTrash() {
		topdir = filepath.Dir(root)
	}
	infoPath := filepath.Join(root, "info", name+infoSuffix)
	original, _ := parseInfo(infoPath, topdir)
	if original == "" {
		return "", errors.New("the original location is unknown")
	}
	dir := filepath.Dir(original)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	target := filepath.Join(dir, UniqueName(dir, filepath.Base(original)))
	if err := os.Rename(path, target); err != nil {
		if !errors.Is(err, syscall.EXDEV) {
			return "", err
		}
		if err := copyTree(path, target); err != nil {
			os.RemoveAll(target)
			return "", err
		}
		if err := os.RemoveAll(path); err != nil {
			return target, err
		}
	}
	os.Remove(infoPath)
	return target, nil
}

func DeleteTrashed(path string) error {
	root, err := trashRootOf(path)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	os.Remove(filepath.Join(root, "info", filepath.Base(path)+infoSuffix))
	return nil
}

func EmptyTrash() (int, error) {
	count := 0
	var firstErr error
	for _, root := range trashRoots() {
		files, err := os.ReadDir(filepath.Join(root, "files"))
		if err != nil {
			continue
		}
		for _, f := range files {
			if err := os.RemoveAll(filepath.Join(root, "files", f.Name())); err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			os.Remove(filepath.Join(root, "info", f.Name()+infoSuffix))
			count++
		}
	}
	return count, firstErr
}

func InTrash(path string) bool {
	_, err := trashRootOf(path)
	return err == nil
}
