package fsutil

import (
	"path/filepath"
	"strings"
)

type Kind string

const (
	KindFolder  Kind = "folder"
	KindImage   Kind = "image"
	KindVideo   Kind = "video"
	KindAudio   Kind = "audio"
	KindDoc     Kind = "doc"
	KindText    Kind = "text"
	KindCode    Kind = "code"
	KindArchive Kind = "archive"
	KindOther   Kind = "other"
)

func setOf(items ...string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, it := range items {
		m[it] = true
	}
	return m
}

var (
	imageExts = setOf("jpg", "jpeg", "jpe", "png", "gif", "webp", "bmp", "tif", "tiff", "svg", "avif", "heic", "heif", "ico", "jxl", "jfif", "apng")
	videoExts = setOf("mp4", "mkv", "webm", "mov", "avi", "m4v", "flv", "wmv", "mpg", "mpeg", "ts", "m2ts", "3gp", "ogv", "vob", "divx")
	audioExts = setOf("mp3", "flac", "wav", "ogg", "oga", "opus", "m4a", "aac", "wma", "aiff", "mka", "amr")
	docExts   = setOf("pdf", "doc", "docx", "odt", "xls", "xlsx", "ods", "ppt", "pptx", "odp", "rtf", "epub", "djvu", "mobi")
	textExts  = setOf("txt", "md", "markdown", "log", "csv", "tsv", "json", "yaml", "yml", "toml", "ini", "conf", "cfg", "xml", "rst", "nfo", "srt", "vtt", "desktop", "service", "env", "tex")
	codeExts  = setOf("go", "py", "js", "mjs", "ts", "tsx", "jsx", "c", "h", "cpp", "cc", "hpp", "cs", "rs", "java", "kt", "sh", "bash", "zsh", "fish", "rb", "php", "lua", "sql", "html", "htm", "css", "scss", "sass", "vue", "svelte", "swift", "pl", "r", "asm", "s", "cmake", "mk", "gradle", "dart")
	archExts  = setOf("zip", "tar", "gz", "tgz", "bz2", "xz", "zst", "7z", "rar", "iso", "deb", "rpm", "appimage", "jar", "cab", "lz4", "img")

	nativeImage = setOf("jpg", "jpeg", "jpe", "png", "gif", "webp", "bmp", "svg", "avif", "ico", "jfif", "apng")
	nativeVideo = setOf("mp4", "m4v", "webm", "ogv", "mov")
	nativeAudio = setOf("mp3", "flac", "wav", "ogg", "oga", "opus", "m4a", "aac")
)

func Ext(name string) string {
	e := strings.ToLower(filepath.Ext(name))
	if len(e) > 1 && name != e {
		return e[1:]
	}
	return ""
}

func KindOf(ext string) Kind {
	switch {
	case imageExts[ext]:
		return KindImage
	case videoExts[ext]:
		return KindVideo
	case audioExts[ext]:
		return KindAudio
	case docExts[ext]:
		return KindDoc
	case codeExts[ext]:
		return KindCode
	case textExts[ext]:
		return KindText
	case archExts[ext]:
		return KindArchive
	}
	return KindOther
}

func BrowserNative(ext string, kind Kind) bool {
	switch kind {
	case KindImage:
		return nativeImage[ext]
	case KindVideo:
		return nativeVideo[ext]
	case KindAudio:
		return nativeAudio[ext]
	}
	return false
}

func Viewable(kind Kind) bool {
	switch kind {
	case KindImage, KindVideo, KindAudio, KindDoc, KindText, KindCode:
		return true
	}
	return false
}
