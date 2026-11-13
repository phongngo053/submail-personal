package icon

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"io"
)

// icoSizes are the resolutions stored in the generated application icon.
var icoSizes = []int{16, 24, 32, 48, 64, 128, 256}

// icoHeader is the ICONDIR structure at the start of an .ico file.
type icoHeader struct {
	Reserved uint16
	Type     uint16
	Count    uint16
}

// icoEntry describes one image inside an .ico file.
type icoEntry struct {
	Width      uint8
	Height     uint8
	ColorCount uint8
	Reserved   uint8
	Planes     uint16
	BitCount   uint16
	BytesInRes uint32
	Offset     uint32
}

// WriteICO writes a multi resolution Windows icon file. Every entry holds a
// PNG, which Windows 10 and 11 read without any conversion.
func WriteICO(w io.Writer, src image.Image, sizes []int) error {
	if len(sizes) == 0 {
		sizes = icoSizes
	}

	images := make([][]byte, 0, len(sizes))
	for _, size := range sizes {
		img := Resize(src, size)
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return fmt.Errorf("icon: không mã hoá được ảnh %dx%d: %w", size, size, err)
		}
		images = append(images, buf.Bytes())
	}

	// The directory sits at the front of the file, so the image data starts
	// right after the header and the entries.
	offset := uint32(6 + 16*len(images))

	entries := make([]icoEntry, 0, len(images))
	for i, data := range images {
		size := sizes[i]
		// A width or height of 256 is stored as 0 in the directory.
		dim := uint8(size)
		if size >= 256 {
			dim = 0
		}
		entries = append(entries, icoEntry{
			Width:      dim,
			Height:     dim,
			ColorCount: 0,
			Reserved:   0,
			Planes:     1,
			BitCount:   32,
			BytesInRes: uint32(len(data)),
			Offset:     offset,
		})
		offset += uint32(len(data))
	}

	if err := binary.Write(w, binary.LittleEndian, icoHeader{
		Reserved: 0,
		Type:     1,
		Count:    uint16(len(images)),
	}); err != nil {
		return fmt.Errorf("icon: không ghi được header ICO: %w", err)
	}
	for _, entry := range entries {
		if err := binary.Write(w, binary.LittleEndian, entry); err != nil {
			return fmt.Errorf("icon: không ghi được mục ICO: %w", err)
		}
	}
	for _, data := range images {
		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("icon: không ghi được dữ liệu ICO: %w", err)
		}
	}
	return nil
}
