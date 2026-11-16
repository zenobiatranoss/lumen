package fsutil

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Entry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Kind    Kind   `json:"kind"`
	Ext     string `json:"ext,omitempty"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mtime"`
	Hidden  bool   `json:"hidden,omitempty"`
	Link    bool   `json:"link,omitempty"`
	Broken  bool   `json:"broken,omitempty"`
	Native  bool   `json:"native,omitempty"`
	W       int    `json:"w,omitempty"`
	H       int    `json:"h,omitempty"`
}

func EntryFromInfo(path string, info fs.FileInfo) Entry {
	name := info.Name()
	e := Entry{
		Name:    name,
		Path:    path,
		ModTime: info.ModTime().UnixMilli(),
		Hidden:  strings.HasPrefix(name, "."),
		Kind:    KindOther,
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		e.Link = true
		target, err := os.Stat(path)
		if err != nil {
			e.Broken = true
			return e
		}
		info = target
		e.ModTime = info.ModTime().UnixMilli()
	}
	if info.IsDir() {
		e.Kind = KindFolder
		return e
	}
	if !info.Mode().IsRegular() {
		return e
	}
	e.Size = info.Size()
	e.Ext = Ext(name)
	e.Kind = KindOf(e.Ext)
	e.Native = BrowserNative(e.Ext, e.Kind)
	return e
}

func Stat(path string) (Entry, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Entry{}, err
	}
	return EntryFromInfo(path, info), nil
}

func ListDir(dir string, showHidden bool) ([]Entry, error) {
	items, err := os.ReadDir(dir)
	if err != nil && len(items) == 0 {
		return nil, err
	}
	entries := make([]Entry, 0, len(items))
	for _, item := range items {
		name := item.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		info, err := item.Info()
		if err != nil {
			continue
		}
		entries = append(entries, EntryFromInfo(filepath.Join(dir, name), info))
	}
	return entries, nil
}

func FirstImages(dir string, limit int) ([]string, int) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0
	}
	var found []string
	for _, item := range items {
		name := item.Name()
		if strings.HasPrefix(name, ".") || item.IsDir() {
			continue
		}
		if KindOf(Ext(name)) == KindImage && len(found) < limit {
			found = append(found, filepath.Join(dir, name))
		}
	}
	return found, len(items)
}
