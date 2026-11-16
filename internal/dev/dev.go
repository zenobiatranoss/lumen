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
