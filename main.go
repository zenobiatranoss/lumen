package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"lumen/internal/config"
	"lumen/internal/media"
	"lumen/internal/search"
	"lumen/internal/server"
	"lumen/web"
)

const version = "1.0.0"

type instance struct {
	PID   int    `json:"pid"`
	Port  int    `json:"port"`
	Token string `json:"token"`
}

func usage() {
	fmt.Fprintf(os.Stderr, `Lumen %s - a calm, image-first file browser

Usage:
  lumen [options] [folder or file]

Options:
`, version)
	flag.PrintDefaults()
}

func main() {
	port := flag.Int("port", 0, "port to listen on, 0 picks a free one")
	noOpen := flag.Bool("no-open", false, "print the address instead of opening a browser")
	readOnly := flag.Bool("readonly", false, "disable every action that changes files")
	idle := flag.Duration("idle", 10*time.Minute, "quit after this long without an open tab, 0 keeps it running")
	browserMode := flag.Bool("browser", false, "open in the default browser instead of a standalone window")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		fmt.Println("lumen", version)
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fatal(err)
	}
	params, err := launchParams(flag.Arg(0))
	if err != nil {
		fatal(err)
	}

	instancePath := filepath.Join(config.Dir(), "instance.json")
	if running, ok := findRunning(instancePath); ok {
		address := buildURL(running.Port, running.Token, params)
		fmt.Println("Lumen is already running:", address)
		if !*noOpen {
			openWindow(address, !*browserMode)
		}
		return
	}

	store, err := config.Open(filepath.Join(config.Dir(), "config.json"), home)
	if err != nil {
		fatal(err)
	}
	cfg := store.Get()

	index := search.NewIndex()
	index.Configure(cfg.IndexRoots, cfg.SkipDirs)
	index.Rebuild()

	token, err := randomToken()
	if err != nil {
		fatal(err)
	}

	srv := server.New(server.Options{
		Home:     home,
		Config:   store,
		Index:    index,
		Thumbs:   media.NewThumbs(filepath.Join(config.CacheDir(), "thumbs"), media.DetectTools()),
		Web:      web.Files,
		Token:    token,
		ReadOnly: *readOnly,
		Version:  version,
	})

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		fatal(err)
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port
	srv.SetPort(actualPort)

	writeInstance(instancePath, instance{PID: os.Getpid(), Port: actualPort, Token: token})
	defer os.Remove(instancePath)

	address := buildURL(actualPort, token, params)
	fmt.Println("Lumen", version, "is running at", address)
	var window *exec.Cmd
	if !*noOpen {
		window = openWindow(address, !*browserMode)
	}

	httpServer := &http.Server{Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintln(os.Stderr, "server stopped:", err)
			stop()
		}
	}()
	go refreshIndex(ctx, index)
	if window != nil {
		go func() {
			started := time.Now()
			window.Wait()
			if time.Since(started) > 3*time.Second {
				stop()
			}
		}()
	}
	if *idle > 0 && window == nil {
		go watchIdle(ctx, srv, *idle, stop)
	}

	<-ctx.Done()
	index.Stop()
	shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdown); err != nil {
		httpServer.Close()
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "lumen:", err)
	os.Exit(1)
}

func launchParams(arg string) (url.Values, error) {
	params := url.Values{}
	if arg == "" {
		return params, nil
	}
	abs, err := filepath.Abs(arg)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		params.Set("path", abs)
		return params, nil
	}
	params.Set("path", filepath.Dir(abs))
	params.Set("view", filepath.Base(abs))
	return params, nil
}

func buildURL(port int, token string, params url.Values) string {
	q := url.Values{}
	for k, v := range params {
		q[k] = v
	}
	q.Set("t", token)
	return fmt.Sprintf("http://127.0.0.1:%d/?%s", port, q.Encode())
}

func randomToken() (string, error) {
	buf := make([]byte, 24)
	f, err := os.Open("/dev/urandom")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Read(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", buf), nil
}

func writeInstance(path string, in instance) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	data, err := json.Marshal(in)
	if err != nil {
		return
	}
	os.WriteFile(path, data, 0o600)
}

func findRunning(path string) (instance, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return instance{}, false
	}
	var in instance
	if json.Unmarshal(data, &in) != nil || in.Port == 0 {
		return instance{}, false
	}
	if syscall.Kill(in.PID, 0) != nil {
		return instance{}, false
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", in.Port), 300*time.Millisecond)
	if err != nil {
		return instance{}, false
	}
	conn.Close()
	return in, true
}

func findAppBrowser() string {
	for _, name := range []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable", "brave-browser", "microsoft-edge", "vivaldi"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
}

func openWindow(address string, appMode bool) *exec.Cmd {
	if appMode {
		if cmd := nativeWindow(address); cmd != nil {
			return cmd
		}
		if bin := findAppBrowser(); bin != "" {
			cmd := exec.Command(bin,
				"--app="+address,
				"--class=lumen",
				"--name=lumen",
				"--ozone-platform=x11",
				"--user-data-dir="+filepath.Join(config.Dir(), "app-profile"),
				"--no-first-run",
				"--no-default-browser-check",
				"--window-size=1440,900",
			)
			cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
			if err := cmd.Start(); err == nil {
				return cmd
			}
		} else {
			fmt.Fprintln(os.Stderr, "no Chromium-based browser found for the app window, install one with: sudo apt install chromium")
		}
	}
	cmd := exec.Command("xdg-open", address)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "could not open a browser, open the address above yourself")
		return nil
	}
	go cmd.Wait()
	return nil
}

func refreshIndex(ctx context.Context, index *search.Index) {
	ticker := time.NewTicker(20 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			index.Rebuild()
		}
	}
}

func watchIdle(ctx context.Context, srv *server.Server, limit time.Duration, stop context.CancelFunc) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if srv.IdleFor() > limit {
				fmt.Println("no open tab for", limit, "- shutting down")
				stop()
				return
			}
		}
	}
}

func nativeWindow(address string) *exec.Cmd {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}
	script := filepath.Join(dataHome, "lumen", "window.py")
	if _, err := os.Stat(script); err != nil {
		exe, err := os.Executable()
		if err != nil {
			return nil
		}
		script = filepath.Join(filepath.Dir(exe), "packaging", "window.py")
		if _, err := os.Stat(script); err != nil {
			return nil
		}
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		return nil
	}
	if exec.Command(python, script, "--check").Run() != nil {
		return nil
	}
	icon := filepath.Join(dataHome, "icons", "hicolor", "256x256", "apps", "lumen.png")
	cmd := exec.Command(python, script, address, icon)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return nil
	}
	return cmd
}
