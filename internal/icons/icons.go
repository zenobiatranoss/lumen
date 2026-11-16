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
