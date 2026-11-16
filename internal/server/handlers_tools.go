package server

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"lumen/internal/fsutil"
	icontheme "lumen/internal/icons"
)

func (s *Server) extraOps(req opRequest, paths []string) []opItem {
	idx := s.opts.Index
	var results []opItem
	add := func(from, to string, err error) {
		item := opItem{From: from, To: to}
		if err != nil {
			item.Error = friendly(err)
		}
		results = append(results, item)
	}
	switch req.Action {
	case "compress":
		out, err := fsutil.Compress(paths, filepath.Dir(paths[0]), req.Name, req.Format)
		if err == nil {
			idx.Add(out)
		}
		add(paths[0], out, err)
	case "extract":
		for _, p := range paths {
			out, err := fsutil.Extract(p, filepath.Dir(p))
			if err == nil {
				idx.Add(out)
			}
			add(p, out, err)
		}
	case "newfile":
		out, err := fsutil.NewFile(paths[0], req.Name)
		if err == nil {
			idx.Add(out)
		}
		add(paths[0], out, err)
	case "duplicate":
		for _, p := range paths {
			out, err := fsutil.Copy(p, filepath.Dir(p))
			if err == nil {
				idx.Add(out)
			}
			add(p, out, err)
		}
	case "symlink":
		name := req.Name
		if name == "" {
			name = filepath.Base(paths[0]) + " link"
		}
		out, err := fsutil.Symlink(paths[0], filepath.Dir(paths[0]), name)
		if err == nil {
			idx.Add(out)
		}
		add(paths[0], out, err)
	case "chmod":
		bits, err := strconv.ParseUint(req.Mode, 8, 32)
		if err != nil || bits > 0o7777 {
			add(paths[0], "", errors.New("the permissions are not valid"))
			break
		}
		for _, p := range paths {
			add(p, "", os.Chmod(p, fs.FileMode(bits)))
		}
	case "bulkrename":
		width := max(2, len(strconv.Itoa(len(paths))))
		for i, p := range paths {
			to, err := fsutil.Rename(p, bulkName(p, req, i+1, width))
			if err == nil && to != p {
				idx.Remove(p)
				idx.Add(to)
			}
			add(p, to, err)
		}
	}
	return results
}

func bulkName(path string, req opRequest, n, width int) string {
	name := filepath.Base(path)
	if req.Find != "" {
		return strings.ReplaceAll(name, req.Find, req.Replace)
	}
	ext := ""
	if info, err := os.Lstat(path); err == nil && !info.IsDir() {
		ext = filepath.Ext(name)
	}
	stem := strings.TrimSuffix(name, ext)
	out := strings.NewReplacer(
		"{name}", stem,
		"{n}", fmt.Sprintf("%0*d", width, n),
		"{ext}", strings.TrimPrefix(ext, "."),
	).Replace(req.Name)
	if !strings.Contains(req.Name, "{ext}") {
		out += ext
	}
	return out
}

type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}

func (s *Server) checksum(w http.ResponseWriter, r *http.Request) {
	path, err := s.pathParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		fail(w, err)
		return
	}
	if info.IsDir() {
		writeError(w, http.StatusBadRequest, "checksums are only available for files")
		return
	}
	f, err := os.Open(path)
	if err != nil {
		fail(w, err)
		return
	}
	defer f.Close()
	m5, s1, s256 := md5.New(), sha1.New(), sha256.New()
	if _, err := io.Copy(io.MultiWriter(m5, s1, s256), ctxReader{r.Context(), f}); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"size":   info.Size(),
		"md5":    hex.EncodeToString(m5.Sum(nil)),
		"sha1":   hex.EncodeToString(s1.Sum(nil)),
		"sha256": hex.EncodeToString(s256.Sum(nil)),
	})
}

type desktopApp struct {
	ID       string
	Name     string
	Exec     string
	Icon     string
	Terminal bool
	Mimes    map[string]bool
}

func appDirs() []string {
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		home, _ := os.UserHomeDir()
		data = filepath.Join(home, ".local", "share")
	}
	dirs := []string{filepath.Join(data, "applications")}
	system := os.Getenv("XDG_DATA_DIRS")
	if system == "" {
		system = "/usr/local/share:/usr/share"
	}
	for _, d := range strings.Split(system, ":") {
		if d != "" {
			dirs = append(dirs, filepath.Join(d, "applications"))
		}
	}
	return append(dirs, "/var/lib/flatpak/exports/share/applications")
}

func parseDesktop(path, id string) (desktopApp, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return desktopApp{}, false
	}
	app := desktopApp{ID: id, Mimes: map[string]bool{}}
	inEntry, isApp, hidden := false, false, false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			inEntry = line == "[Desktop Entry]"
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !inEntry || !ok {
			continue
		}
		switch key {
		case "Name":
			app.Name = value
		case "Exec":
			app.Exec = value
		case "Icon":
			app.Icon = value
		case "Type":
			isApp = value == "Application"
		case "NoDisplay", "Hidden":
			hidden = hidden || value == "true"
		case "Terminal":
			app.Terminal = value == "true"
		case "MimeType":
			for _, m := range strings.Split(value, ";") {
				if m != "" {
					app.Mimes[m] = true
				}
			}
		}
	}
	return app, isApp && !hidden && app.Name != "" && app.Exec != ""
}

func loadApps() []desktopApp {
	seen := map[string]bool{}
	var apps []desktopApp
	for _, dir := range appDirs() {
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".desktop") {
				return nil
			}
			rel, _ := filepath.Rel(dir, path)
			id := strings.ReplaceAll(rel, string(os.PathSeparator), "-")
			if seen[id] {
				return nil
			}
			seen[id] = true
			if app, ok := parseDesktop(path, id); ok {
				apps = append(apps, app)
			}
			return nil
		})
	}
	sort.Slice(apps, func(i, j int) bool { return strings.ToLower(apps[i].Name) < strings.ToLower(apps[j].Name) })
	return apps
}

func mimeFor(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "xdg-mime", "query", "filetype", path).Output(); err == nil {
		if m := strings.TrimSpace(string(out)); m != "" {
			return m
		}
	}
	if m := mime.TypeByExtension(strings.ToLower(filepath.Ext(path))); m != "" {
		return strings.SplitN(m, ";", 2)[0]
	}
	return "application/octet-stream"
}

func defaultAppID(mimeType string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "xdg-mime", "query", "default", mimeType).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

type appRow struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

func (s *Server) openWithList(w http.ResponseWriter, r *http.Request) {
	path, err := s.pathParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	mt := mimeFor(path)
	def := defaultAppID(mt)
	recommended, others := []appRow{}, []appRow{}
	for _, a := range loadApps() {
		icon := a.Icon
		if !icontheme.ValidName(icon) {
			icon = "application-x-executable"
		}
		row := appRow{ID: a.ID, Name: a.Name, Icon: icon}
		match := a.Mimes[mt] || (strings.HasPrefix(mt, "text/") && a.Mimes["text/plain"])
		switch {
		case a.ID == def:
			recommended = append([]appRow{row}, recommended...)
		case match:
			recommended = append(recommended, row)
		default:
			others = append(others, row)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"mime": mt, "default": def, "recommended": recommended, "others": others})
}

func splitExec(line string) []string {
	var parts []string
	var cur strings.Builder
	quote := rune(0)
	started := false
	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		switch {
		case c == '\\' && i+1 < len(runes):
			i++
			cur.WriteRune(runes[i])
			started = true
		case quote != 0:
			if c == quote {
				quote = 0
			} else {
				cur.WriteRune(c)
			}
		case c == '"' || c == '\'':
			quote = c
			started = true
		case c == ' ' || c == '\t':
			if started {
				parts = append(parts, cur.String())
				cur.Reset()
				started = false
			}
		default:
			cur.WriteRune(c)
			started = true
		}
	}
	if started {
		parts = append(parts, cur.String())
	}
	return parts
}

func expandExec(line, path string) []string {
	var out []string
	used := false
	for _, p := range splitExec(line) {
		switch p {
		case "%i", "%c", "%k", "%d", "%D", "%n", "%N", "%v", "%m":
			continue
		case "%f", "%F", "%u", "%U":
			out = append(out, path)
			used = true
			continue
		}
		for _, code := range []string{"%f", "%F", "%u", "%U"} {
			if strings.Contains(p, code) {
				p = strings.ReplaceAll(p, code, path)
				used = true
			}
		}
		out = append(out, strings.ReplaceAll(p, "%%", "%"))
	}
	if !used {
		out = append(out, path)
	}
	return out
}

func startDetached(dir string, argv ...string) error {
	if len(argv) == 0 {
		return errors.New("empty command")
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

func terminalCommand(dir string) []string {
	for _, name := range []string{"konsole", "gnome-terminal", "xfce4-terminal", "mate-terminal", "alacritty", "kitty", "foot", "x-terminal-emulator", "xterm"} {
		bin, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		switch name {
		case "konsole":
			return []string{bin, "--workdir", dir}
		case "alacritty":
			return []string{bin, "--working-directory", dir}
		case "gnome-terminal", "xfce4-terminal", "mate-terminal", "foot":
			return []string{bin, "--working-directory=" + dir}
		case "kitty":
			return []string{bin, "--directory", dir}
		}
		return []string{bin}
	}
	return nil
}

func (s *Server) openWithRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path        string `json:"path"`
		App         string `json:"app"`
		MakeDefault bool   `json:"makeDefault"`
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
	var app *desktopApp
	for _, a := range loadApps() {
		if a.ID == req.App {
			chosen := a
			app = &chosen
			break
		}
	}
	if app == nil {
		writeError(w, http.StatusBadRequest, "that application is not available")
		return
	}
	argv := expandExec(app.Exec, path)
	dir := filepath.Dir(path)
	if app.Terminal {
		if term := terminalCommand(dir); term != nil {
			flag := "-e"
			if filepath.Base(term[0]) == "gnome-terminal" {
				flag = "--"
			}
			argv = append(append(term, flag), argv...)
		}
	}
	if err := startDetached(dir, argv...); err != nil {
		writeError(w, http.StatusInternalServerError, "could not start "+app.Name)
		return
	}
	if req.MakeDefault {
		exec.Command("xdg-mime", "default", app.ID, mimeFor(path)).Run()
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) terminal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
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
	if info, err := os.Stat(path); err != nil {
		fail(w, err)
		return
	} else if !info.IsDir() {
		path = filepath.Dir(path)
	}
	argv := terminalCommand(path)
	if argv == nil {
		writeError(w, http.StatusNotFound, "no terminal application was found")
		return
	}
	if err := startDetached(path, argv...); err != nil {
		writeError(w, http.StatusInternalServerError, "could not open a terminal")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
