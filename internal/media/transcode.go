package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
)

type flushWriter struct {
	w io.Writer
}

func (f flushWriter) Write(p []byte) (int, error) {
	n, err := f.w.Write(p)
	if fl, ok := f.w.(http.Flusher); ok {
		fl.Flush()
	}
	return n, err
}

func (t Tools) Stream(ctx context.Context, w io.Writer, path string, start float64, p *Probe) error {
	if t.FFmpeg == "" {
		return errors.New("ffmpeg is not installed")
	}
	args := []string{"-v", "error", "-nostdin"}
	if start > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.2f", start))
	}
	args = append(args, "-i", path, "-map", "0:v:0", "-map", "0:a:0?", "-sn")
	if p != nil && p.VideoCodec == "h264" && p.PixFmt == "yuv420p" {
		args = append(args, "-c:v", "copy")
	} else {
		args = append(args,
			"-c:v", "libx264", "-preset", "veryfast", "-crf", "23",
			"-pix_fmt", "yuv420p",
			"-vf", "scale='min(1920,iw)':-2",
		)
	}
	if p != nil && p.AudioCodec == "aac" {
		args = append(args, "-c:a", "copy")
	} else {
		args = append(args, "-c:a", "aac", "-b:a", "160k", "-ac", "2")
	}
	args = append(args,
		"-movflags", "frag_keyframe+empty_moov+default_base_moof",
		"-f", "mp4", "pipe:1",
	)
	cmd := exec.CommandContext(ctx, t.FFmpeg, args...)
	cmd.Stdout = flushWriter{w}
	err := cmd.Run()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
