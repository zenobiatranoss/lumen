package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type Favorite struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Config struct {
	Favorites  []Favorite     `json:"favorites"`
	IndexRoots []string       `json:"indexRoots"`
	SkipDirs   []string       `json:"skipDirs"`
	UI         map[string]any `json:"ui"`
}

type Store struct {
	mu   sync.RWMutex
	path string
	cfg  Config
}

func Dir() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "lumen")
}

func CacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "lumen")
}

func defaults(home string) Config {
	return Config{
		Favorites:  []Favorite{},
		IndexRoots: []string{home},
		SkipDirs:   []string{"node_modules", "__pycache__", "venv", "site-packages"},
		UI:         map[string]any{},
	}
}

func Open(path, home string) (*Store, error) {
	s := &Store{path: path, cfg: defaults(home)}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		return s, nil
	}
	s.cfg = normalize(loaded, home)
	return s, nil
}

func normalize(c Config, home string) Config {
	d := defaults(home)
	if c.Favorites == nil {
		c.Favorites = d.Favorites
	}
	if len(c.IndexRoots) == 0 {
		c.IndexRoots = d.IndexRoots
	}
	if c.SkipDirs == nil {
		c.SkipDirs = d.SkipDirs
	}
	if c.UI == nil {
		c.UI = d.UI
	}
	favs := make([]Favorite, 0, len(c.Favorites))
	for _, f := range c.Favorites {
		if filepath.IsAbs(f.Path) {
			f.Path = filepath.Clean(f.Path)
			if f.Name == "" {
				f.Name = filepath.Base(f.Path)
			}
			favs = append(favs, f)
		}
	}
	c.Favorites = favs
	roots := make([]string, 0, len(c.IndexRoots))
	for _, r := range c.IndexRoots {
		if filepath.IsAbs(r) {
			roots = append(roots, filepath.Clean(r))
		}
	}
	if len(roots) == 0 {
		roots = d.IndexRoots
	}
	c.IndexRoots = roots
	return c
}

func (s *Store) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := Config{
		Favorites:  append([]Favorite{}, s.cfg.Favorites...),
		IndexRoots: append([]string{}, s.cfg.IndexRoots...),
		SkipDirs:   append([]string{}, s.cfg.SkipDirs...),
		UI:         make(map[string]any, len(s.cfg.UI)),
	}
	for k, v := range s.cfg.UI {
		out.UI[k] = v
	}
	return out
}

func (s *Store) Replace(c Config, home string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = normalize(c, home)
	return s.save()
}

func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
