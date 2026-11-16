package media

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

type EXIF struct {
	Orientation int    `json:"orientation,omitempty"`
	Make        string `json:"make,omitempty"`
	Model       string `json:"model,omitempty"`
	Lens        string `json:"lens,omitempty"`
	Taken       string `json:"taken,omitempty"`
	Exposure    string `json:"exposure,omitempty"`
	FNumber     string `json:"fnumber,omitempty"`
	ISO         int    `json:"iso,omitempty"`
	Focal       string `json:"focal,omitempty"`
}

func (e EXIF) Empty() bool {
	return e == EXIF{}
}

type ifdEntry struct {
	tag   uint16
	typ   uint16
	count uint32
	data  []byte
	order binary.ByteOrder
}

func typeSize(typ uint16) int {
	switch typ {
	case 1, 2, 6, 7:
		return 1
	case 3, 8:
		return 2
	case 4, 9, 11:
		return 4
	case 5, 10, 12:
		return 8
	}
	return 0
}

func readIFD(b []byte, order binary.ByteOrder, off uint32) []ifdEntry {
	if int(off)+2 > len(b) {
		return nil
	}
	n := int(order.Uint16(b[off:]))
	entries := make([]ifdEntry, 0, n)
	for i := 0; i < n; i++ {
		p := int(off) + 2 + i*12
		if p+12 > len(b) {
			break
		}
		typ := order.Uint16(b[p+2:])
		count := order.Uint32(b[p+4:])
		unit := typeSize(typ)
		if unit == 0 || count > 1<<16 {
			continue
		}
		size := unit * int(count)
		var data []byte
		if size <= 4 {
			data = b[p+8 : p+8+size]
		} else {
			start := int(order.Uint32(b[p+8:]))
			if start < 0 || start+size > len(b) {
				continue
			}
			data = b[start : start+size]
		}
		entries = append(entries, ifdEntry{tag: order.Uint16(b[p:]), typ: typ, count: count, data: data, order: order})
	}
	return entries
}

func (e ifdEntry) text() string {
	return strings.TrimSpace(strings.TrimRight(string(e.data), "\x00"))
}

func (e ifdEntry) integer() int {
	switch {
	case e.typ == 3 && len(e.data) >= 2:
		return int(e.order.Uint16(e.data))
	case e.typ == 4 && len(e.data) >= 4:
		return int(e.order.Uint32(e.data))
	}
	return 0
}

func (e ifdEntry) rational() (uint32, uint32, bool) {
	if e.typ != 5 || len(e.data) < 8 {
		return 0, 0, false
	}
	den := e.order.Uint32(e.data[4:])
	if den == 0 {
		return 0, 0, false
	}
	return e.order.Uint32(e.data), den, true
}

func parseTIFF(b []byte) (EXIF, bool) {
	var ex EXIF
	if len(b) < 8 {
		return ex, false
	}
	var order binary.ByteOrder
	switch string(b[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return ex, false
	}
	if order.Uint16(b[2:]) != 42 {
		return ex, false
	}
	var exifPointer uint32
	for _, e := range readIFD(b, order, order.Uint32(b[4:])) {
		switch e.tag {
		case 0x010F:
			ex.Make = e.text()
		case 0x0110:
			ex.Model = e.text()
		case 0x0112:
			if o := e.integer(); o >= 1 && o <= 8 {
				ex.Orientation = o
			}
		case 0x8769:
			exifPointer = uint32(e.integer())
		}
	}
	if exifPointer != 0 {
		for _, e := range readIFD(b, order, exifPointer) {
			switch e.tag {
			case 0x829A:
				if n, d, ok := e.rational(); ok && n > 0 {
					if n >= d {
						ex.Exposure = fmt.Sprintf("%.1fs", float64(n)/float64(d))
					} else {
						ex.Exposure = fmt.Sprintf("1/%ds", int(float64(d)/float64(n)+0.5))
					}
				}
			case 0x829D:
				if n, d, ok := e.rational(); ok {
					ex.FNumber = fmt.Sprintf("f/%.1f", float64(n)/float64(d))
				}
			case 0x8827:
				ex.ISO = e.integer()
			case 0x9003:
				ex.Taken = e.text()
			case 0x920A:
				if n, d, ok := e.rational(); ok {
					ex.Focal = fmt.Sprintf("%.0f mm", float64(n)/float64(d))
				}
			case 0xA434:
				ex.Lens = e.text()
			}
		}
	}
	return ex, true
}

func ReadEXIF(path string) (EXIF, bool) {
	f, err := os.Open(path)
	if err != nil {
		return EXIF{}, false
	}
	defer f.Close()
	var soi [2]byte
	if _, err := f.ReadAt(soi[:], 0); err != nil || soi[0] != 0xFF || soi[1] != 0xD8 {
		return EXIF{}, false
	}
	pos := int64(2)
	for i := 0; i < 40; i++ {
		var head [4]byte
		if _, err := f.ReadAt(head[:], pos); err != nil || head[0] != 0xFF {
			return EXIF{}, false
		}
		marker := head[1]
		switch {
		case marker == 0xFF:
			pos++
			continue
		case marker == 0xDA || marker == 0xD9:
			return EXIF{}, false
		case marker == 0x01 || (marker >= 0xD0 && marker <= 0xD8):
			pos += 2
			continue
		}
		size := int64(binary.BigEndian.Uint16(head[2:]))
		if size < 2 {
			return EXIF{}, false
		}
		if marker == 0xE1 && size > 8 {
			buf := make([]byte, size-2)
			n, err := f.ReadAt(buf, pos+4)
			if err != nil && err != io.EOF {
				return EXIF{}, false
			}
			buf = buf[:n]
			if bytes.HasPrefix(buf, []byte("Exif\x00\x00")) {
				return parseTIFF(buf[6:])
			}
		}
		pos += 2 + size
	}
	return EXIF{}, false
}

func Orientation(path string) int {
	if ex, ok := ReadEXIF(path); ok && ex.Orientation > 0 {
		return ex.Orientation
	}
	return 1
}
