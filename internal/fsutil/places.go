package fsutil

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type Place struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Icon string `json:"icon"`
}

type Drive struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	FS    string `json:"fs"`
	Total uint64 `json:"total"`
	Free  uint64 `json:"free"`
}

func userDirs(home string) map[string]string {
	dirs := map[string]string{}
	cfg := os.Getenv("XDG_CONFIG_HOME")
	if cfg == "" {
		cfg = filepath.Join(home, ".config")
	}
	file, err := os.Open(filepath.Join(cfg, "user-dirs.dirs"))
	if err != nil {
		return dirs
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "XDG_") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.Trim(value, `"`)
		value = strings.ReplaceAll(value, "$HOME", home)
		if filepath.IsAbs(value) {
			dirs[key] = filepath.Clean(value)
		}
	}
	return dirs
}

func Places(home string) []Place {
	xdg := userDirs(home)
	pick := func(key, fallback string) string {
		if p, ok := xdg[key]; ok {
			return p
		}
		return filepath.Join(home, fallback)
	}
	candidates := []Place{
		{"Home", home, "home"},
		{"Desktop", pick("XDG_DESKTOP_DIR", "Desktop"), "desktop"},
		{"Documents", pick("XDG_DOCUMENTS_DIR", "Documents"), "documents"},
		{"Downloads", pick("XDG_DOWNLOAD_DIR", "Downloads"), "downloads"},
		{"Pictures", pick("XDG_PICTURES_DIR", "Pictures"), "pictures"},
		{"Videos", pick("XDG_VIDEOS_DIR", "Videos"), "videos"},
		{"Music", pick("XDG_MUSIC_DIR", "Music"), "music"},
	}
	places := make([]Place, 0, len(candidates)+1)
	for _, p := range candidates {
		if p.Icon == "home" {
			places = append(places, p)
			continue
		}
		if info, err := os.Stat(p.Path); err == nil && info.IsDir() && p.Path != home {
			places = append(places, p)
		}
	}
	places = append(places, Place{"Root", "/", "root"})
	return places
}

func unescapeMount(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) {
			if n, err := strconv.ParseUint(s[i+1:i+4], 8, 8); err == nil {
				b.WriteByte(byte(n))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

var skipFS = setOf("tmpfs", "devtmpfs", "proc", "sysfs", "cgroup", "cgroup2", "squashfs", "overlay", "devpts", "securityfs", "debugfs", "tracefs", "fusectl", "configfs", "pstore", "bpf", "autofs", "mqueue", "hugetlbfs", "binfmt_misc", "efivarfs", "ramfs", "nsfs")

func Drives() []Drive {
	file, err := os.Open("/proc/self/mounts")
	if err != nil {
		return nil
	}
	defer file.Close()
	seen := map[string]bool{}
	var drives []Drive
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		mount := unescapeMount(fields[1])
		fsType := fields[2]
		if skipFS[fsType] || seen[mount] {
			continue
		}
		relevant := mount == "/" ||
			strings.HasPrefix(mount, "/media/") ||
			strings.HasPrefix(mount, "/run/media/") ||
			strings.HasPrefix(mount, "/mnt/")
		if !relevant {
			continue
		}
		var st syscall.Statfs_t
		if err := syscall.Statfs(mount, &st); err != nil {
			continue
		}
		seen[mount] = true
		name := filepath.Base(mount)
		if mount == "/" {
			name = "System"
		}
		drives = append(drives, Drive{
			Name:  name,
			Path:  mount,
			FS:    fsType,
			Total: st.Blocks * uint64(st.Bsize),
			Free:  st.Bavail * uint64(st.Bsize),
		})
	}
	return drives
}

func FreeSpace(path string) (total, free uint64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0
	}
	return st.Blocks * uint64(st.Bsize), st.Bavail * uint64(st.Bsize)
}

var slowFS = setOf("nfs", "nfs4", "cifs", "smbfs", "davfs", "fuse.sshfs", "fuse.rclone", "fuse.gvfsd-fuse", "fuse.portal", "fuse.s3fs")

func SlowMounts() []string {
	file, err := os.Open("/proc/self/mounts")
	if err != nil {
		return nil
	}
	defer file.Close()
	var mounts []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 3 && slowFS[fields[2]] {
			mounts = append(mounts, unescapeMount(fields[1]))
		}
	}
	return mounts
}
