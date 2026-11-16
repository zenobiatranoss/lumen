package web

import "embed"

//go:embed index.html icon.png css js
var Files embed.FS
