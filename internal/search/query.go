package search

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"lumen/internal/fsutil"
)

type Query struct {
	Terms    []string
	Not      []string
	In       []string
	Kinds    map[fsutil.Kind]bool
	Exts     map[string]bool
	TypeExts map[string]bool
	Contains []string
	MinSize  int64
	MaxSize  int64
	After    int64
	Before   int64
	Regex    *regexp.Regexp
}

func (q Query) Empty() bool {
	return len(q.Terms) == 0 && len(q.Not) == 0 && len(q.In) == 0 && len(q.Kinds) == 0 &&
		len(q.Exts) == 0 && len(q.TypeExts) == 0 && len(q.Contains) == 0 && q.MinSize < 0 && q.MaxSize < 0 && q.After == 0 && q.Before == 0 && q.Regex == nil
}

func tokenize(raw string) []string {
	var tokens []string
	var cur strings.Builder
	quoted := false
	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}
	for _, r := range raw {
		switch {
		case r == '"':
			quoted = !quoted
		case (r == ' ' || r == '\t') && !quoted:
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return tokens
}

var kindAliases = map[string][]fsutil.Kind{
	"image": {fsutil.KindImage}, "images": {fsutil.KindImage}, "img": {fsutil.KindImage}, "photo": {fsutil.KindImage}, "photos": {fsutil.KindImage}, "picture": {fsutil.KindImage}, "pictures": {fsutil.KindImage},
	"video": {fsutil.KindVideo}, "videos": {fsutil.KindVideo}, "movie": {fsutil.KindVideo}, "movies": {fsutil.KindVideo},
	"audio": {fsutil.KindAudio}, "music": {fsutil.KindAudio}, "sound": {fsutil.KindAudio},
	"media": {fsutil.KindImage, fsutil.KindVideo},
	"doc":   {fsutil.KindDoc, fsutil.KindText}, "docs": {fsutil.KindDoc, fsutil.KindText}, "document": {fsutil.KindDoc, fsutil.KindText}, "documents": {fsutil.KindDoc, fsutil.KindText},
	"text": {fsutil.KindText}, "code": {fsutil.KindCode}, "source": {fsutil.KindCode},
	"archive": {fsutil.KindArchive}, "archives": {fsutil.KindArchive}, "zip": {fsutil.KindArchive},
	"folder": {fsutil.KindFolder}, "folders": {fsutil.KindFolder}, "dir": {fsutil.KindFolder}, "dirs": {fsutil.KindFolder}, "directory": {fsutil.KindFolder},
	"file":  {fsutil.KindImage, fsutil.KindVideo, fsutil.KindAudio, fsutil.KindDoc, fsutil.KindText, fsutil.KindCode, fsutil.KindArchive, fsutil.KindOther},
	"files": {fsutil.KindImage, fsutil.KindVideo, fsutil.KindAudio, fsutil.KindDoc, fsutil.KindText, fsutil.KindCode, fsutil.KindArchive, fsutil.KindOther},
	"other": {fsutil.KindOther},
}

var programExts = []string{"exe", "msi", "bat", "cmd", "com", "scr", "appimage", "deb", "rpm", "run", "bin", "apk", "dll"}

var programAliases = map[string][]string{
	"exe": programExts, "exes": programExts, "app": programExts, "apps": programExts,
	"program": programExts, "programs": programExts, "executable": programExts,
	"executables": programExts, "installer": programExts, "installers": programExts,
}

var sizeUnits = map[string]float64{
	"": 1, "b": 1,
	"k": 1 << 10, "kb": 1 << 10, "kib": 1 << 10,
	"m": 1 << 20, "mb": 1 << 20, "mib": 1 << 20,
	"g": 1 << 30, "gb": 1 << 30, "gib": 1 << 30,
	"t": 1 << 40, "tb": 1 << 40, "tib": 1 << 40,
}

func parseSize(s string) (int64, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	i := 0
	for i < len(s) && (s[i] == '.' || (s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	if i == 0 {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	n, err := strconv.ParseFloat(s[:i], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	mult, ok := sizeUnits[strings.TrimSpace(s[i:])]
	if !ok {
		return 0, fmt.Errorf("unknown size unit in %q", s)
	}
	return int64(n * mult), nil
}

func parseAge(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid age %q", s)
	}
	n, err := strconv.Atoi(s[:len(s)-1])
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid age %q", s)
	}
	day := 24 * time.Hour
	switch s[len(s)-1] {
	case 'h':
		return time.Duration(n) * time.Hour, nil
	case 'd':
		return time.Duration(n) * day, nil
	case 'w':
		return time.Duration(n) * 7 * day, nil
	case 'm':
		return time.Duration(n) * 30 * day, nil
	case 'y':
		return time.Duration(n) * 365 * day, nil
	}
	return 0, fmt.Errorf("invalid age %q", s)
}

func parseDay(s string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02", "2006-01", "2006"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date %q", s)
}

func dateRange(s string) (time.Time, time.Time, error) {
	start, err := parseDay(s)
	if err != nil {
		return start, start, err
	}
	switch strings.Count(s, "-") {
	case 0:
		return start, start.AddDate(1, 0, 0), nil
	case 1:
		return start, start.AddDate(0, 1, 0), nil
	}
	return start, start.AddDate(0, 0, 1), nil
}

func splitSizeOp(tok string) (op string, value string, ok bool) {
	lower := strings.ToLower(tok)
	if !strings.HasPrefix(lower, "size") {
		return "", "", false
	}
	rest := tok[4:]
	for _, candidate := range []string{">=", "<=", ">", "<", "=", ":"} {
		if strings.HasPrefix(rest, candidate) {
			return candidate, rest[len(candidate):], true
		}
	}
	return "", "", false
}

func Parse(raw string, now time.Time) (Query, error) {
	q := Query{MinSize: -1, MaxSize: -1}
	for _, tok := range tokenize(raw) {
		if op, value, ok := splitSizeOp(tok); ok {
			n, err := parseSize(value)
			if err != nil {
				return q, err
			}
			switch op {
			case ">", ">=":
				q.MinSize = n
			case "<", "<=":
				q.MaxSize = n
			default:
				q.MinSize, q.MaxSize = n, n
			}
			continue
		}
		if strings.HasPrefix(tok, "-") && len(tok) > 1 && !strings.Contains(tok, ":") {
			q.Not = append(q.Not, strings.ToLower(tok[1:]))
			continue
		}
		key, value, hasKey := strings.Cut(tok, ":")
		key = strings.ToLower(key)
		if !hasKey || value == "" {
			q.Terms = append(q.Terms, strings.ToLower(tok))
			continue
		}
		switch key {
		case "type", "kind":
			if q.Kinds == nil {
				q.Kinds = map[fsutil.Kind]bool{}
			}
			for _, part := range strings.Split(strings.ToLower(value), ",") {
				if exts, isProgram := programAliases[strings.TrimSpace(part)]; isProgram {
					if q.TypeExts == nil {
						q.TypeExts = map[string]bool{}
					}
					for _, e := range exts {
						q.TypeExts[e] = true
					}
					continue
				}
				kinds, ok := kindAliases[strings.TrimSpace(part)]
				if !ok {
					return q, fmt.Errorf("unknown type %q", part)
				}
				for _, k := range kinds {
					q.Kinds[k] = true
				}
			}
		case "ext":
			if q.Exts == nil {
				q.Exts = map[string]bool{}
			}
			for _, part := range strings.Split(strings.ToLower(value), ",") {
				q.Exts[strings.TrimPrefix(strings.TrimSpace(part), ".")] = true
			}
		case "newer", "within":
			age, err := parseAge(value)
			if err != nil {
				return q, err
			}
			q.After = now.Add(-age).UnixMilli()
		case "older":
			age, err := parseAge(value)
			if err != nil {
				return q, err
			}
			q.Before = now.Add(-age).UnixMilli()
		case "after":
			t, err := parseDay(value)
			if err != nil {
				return q, err
			}
			q.After = t.UnixMilli()
		case "before":
			t, err := parseDay(value)
			if err != nil {
				return q, err
			}
			q.Before = t.UnixMilli()
		case "date":
			start, end, err := dateRange(value)
			if err != nil {
				return q, err
			}
			q.After, q.Before = start.UnixMilli(), end.UnixMilli()
		case "in", "path":
			q.In = append(q.In, strings.ToLower(value))
		case "re", "regex":
			re, err := regexp.Compile("(?i)" + value)
			if err != nil {
				return q, errors.New("invalid regular expression")
			}
			q.Regex = re
		default:
			q.Terms = append(q.Terms, strings.ToLower(tok))
		}
	}
	return q, nil
}

func (q Query) Match(it Item) bool {
	if len(q.Kinds) > 0 || len(q.TypeExts) > 0 {
		if !q.Kinds[it.Kind] && !(len(q.TypeExts) > 0 && q.TypeExts[fsutil.Ext(it.Name())]) {
			return false
		}
	}
	if q.MinSize >= 0 && (it.Kind == fsutil.KindFolder || it.Size < q.MinSize) {
		return false
	}
	if q.MaxSize >= 0 && (it.Kind == fsutil.KindFolder || it.Size > q.MaxSize) {
		return false
	}
	if q.After != 0 && it.ModTime < q.After {
		return false
	}
	if q.Before != 0 && it.ModTime >= q.Before {
		return false
	}
	name := it.Name()
	lower := strings.ToLower(name)
	if len(q.Exts) > 0 && !q.Exts[fsutil.Ext(lower)] {
		return false
	}
	for _, t := range q.Terms {
		if !strings.Contains(lower, t) {
			return false
		}
	}
	for _, t := range q.Not {
		if strings.Contains(lower, t) {
			return false
		}
	}
	if q.Regex != nil && !q.Regex.MatchString(name) {
		return false
	}
	if len(q.In) > 0 {
		lowerPath := strings.ToLower(it.Path)
		for _, t := range q.In {
			if !strings.Contains(lowerPath, t) {
				return false
			}
		}
	}
	return true
}

func (q Query) MatchContent(it Item) bool {
	if len(q.Contains) == 0 {
		return true
	}
	if (it.Kind != fsutil.KindText && it.Kind != fsutil.KindCode) || it.Size > 2<<20 {
		return false
	}
	data, err := os.ReadFile(it.Path)
	if err != nil || bytes.IndexByte(data[:min(len(data), 4096)], 0) >= 0 {
		return false
	}
	text := strings.ToLower(string(data))
	for _, term := range q.Contains {
		if !strings.Contains(text, term) {
			return false
		}
	}
	return true
}
