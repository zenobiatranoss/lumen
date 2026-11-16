package media

import "os/exec"

type Tools struct {
	FFmpeg   string
	FFprobe  string
	PDFToPPM string
	Convert  string
}

func DetectTools() Tools {
	find := func(names ...string) string {
		for _, name := range names {
			if path, err := exec.LookPath(name); err == nil {
				return path
			}
		}
		return ""
	}
	return Tools{
		FFmpeg:   find("ffmpeg"),
		FFprobe:  find("ffprobe"),
		PDFToPPM: find("pdftoppm"),
		Convert:  find("magick", "convert"),
	}
}
