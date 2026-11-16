package media

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Probe struct {
	Duration   float64 `json:"duration"`
	Width      int     `json:"width,omitempty"`
	Height     int     `json:"height,omitempty"`
	VideoCodec string  `json:"videoCodec,omitempty"`
	AudioCodec string  `json:"audioCodec,omitempty"`
	PixFmt     string  `json:"pixFmt,omitempty"`
	Bitrate    int64   `json:"bitrate,omitempty"`
	FPS        float64 `json:"fps,omitempty"`
	Title      string  `json:"title,omitempty"`
	Artist     string  `json:"artist,omitempty"`
	Album      string  `json:"album,omitempty"`
}

type probeOutput struct {
	Format struct {
		Duration string            `json:"duration"`
		BitRate  string            `json:"bit_rate"`
		Tags     map[string]string `json:"tags"`
	} `json:"format"`
	Streams []struct {
		CodecType   string `json:"codec_type"`
		CodecName   string `json:"codec_name"`
		Width       int    `json:"width"`
		Height      int    `json:"height"`
		PixFmt      string `json:"pix_fmt"`
		AvgFrame    string `json:"avg_frame_rate"`
		Disposition struct {
			AttachedPic int `json:"attached_pic"`
		} `json:"disposition"`
	} `json:"streams"`
}

func parseRate(s string) float64 {
	num, den, ok := strings.Cut(s, "/")
	if !ok {
		return 0
	}
	n, err1 := strconv.ParseFloat(num, 64)
	d, err2 := strconv.ParseFloat(den, 64)
	if err1 != nil || err2 != nil || d == 0 {
		return 0
	}
	return n / d
}

func (t Tools) Probe(ctx context.Context, path string) (*Probe, error) {
	if t.FFprobe == "" {
		return nil, errors.New("ffprobe is not installed")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, t.FFprobe,
		"-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", path).Output()
	if err != nil {
		return nil, err
	}
	var raw probeOutput
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}
	p := &Probe{}
	p.Duration, _ = strconv.ParseFloat(raw.Format.Duration, 64)
	p.Bitrate, _ = strconv.ParseInt(raw.Format.BitRate, 10, 64)
	for key, value := range raw.Format.Tags {
		switch strings.ToLower(key) {
		case "title":
			p.Title = value
		case "artist":
			p.Artist = value
		case "album":
			p.Album = value
		}
	}
	for _, s := range raw.Streams {
		switch s.CodecType {
		case "video":
			if s.Disposition.AttachedPic == 1 || p.VideoCodec != "" {
				continue
			}
			p.VideoCodec = s.CodecName
			p.Width, p.Height = s.Width, s.Height
			p.PixFmt = s.PixFmt
			p.FPS = parseRate(s.AvgFrame)
		case "audio":
			if p.AudioCodec == "" {
				p.AudioCodec = s.CodecName
			}
		}
	}
	return p, nil
}
