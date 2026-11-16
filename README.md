# Lumen
**I built this a long time ago and wanted to share it; this is the file manager I use on my system.**

A calm, image-first file browser written in Go. A single binary serves a local web UI (dark gray theme)
that you open in your browser or as an app window.

Install: `./install.sh`  (needs Go 1.22+)
Run: `lumen [folder or file]` opens as a standalone desktop window. Options: `-browser` `-readonly` `-no-open` `-port N` `-idle 10m`

Optional tools: `sudo apt install ffmpeg poppler-utils imagemagick`
(video thumbnails and playback, PDF thumbnails, HEIC/TIFF support).

Search syntax: `sunset type:image size>2mb newer:30d ext:png,jpg -thumb in:Trip re:^IMG_\d+ date:2026-10`

Local only: listens on 127.0.0.1, random token per run, host and origin checks, deletions go to the XDG trash.
