// Package icon turns generated artwork into the PNG and ICO files the app
// uses, and builds the prompts that describe the wanted picture.
//
// The image service itself lives in the integrations package: it is optional
// and configured in a local file that never leaves the machine.
package icon

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

// Errors reported to the user. The frontend maps them to Vietnamese toasts.
var (
	ErrEmptyPrompt = errors.New("Prompt tạo icon không được để trống")
	ErrBadImage    = errors.New("Ảnh trả về không phải định dạng ảnh hợp lệ")
	ErrBadName     = errors.New("Tên file không hợp lệ")
)

// DecodePNG decodes image bytes into an image.
//
// The image service is not consistent about the format: it sometimes answers
// with a PNG and sometimes with a JPEG even though the URL ends in .png, so
// both are accepted here and the result is stored as PNG.
func DecodePNG(data []byte) (image.Image, error) {
	reader := bytes.NewReader(data)
	if img, err := png.Decode(reader); err == nil {
		return img, nil
	}
	reader.Reset(data)
	if img, err := jpeg.Decode(reader); err == nil {
		return img, nil
	}
	return nil, fmt.Errorf("%w: không giải mã được ảnh PNG hoặc JPEG", ErrBadImage)
}

// Resize scales an image to a square of the given size.
func Resize(src image.Image, size int) image.Image {
	if size <= 0 {
		return src
	}
	bounds := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	return dst
}

// SavePNG writes an image as a PNG file. The write is atomic: the bytes go to
// a temporary file in the same folder and are then renamed over the target, so
// an interrupted write cannot leave a truncated icon behind.
func SavePNG(dir, name string, img image.Image) error {
	if !ValidName(name) {
		return ErrBadName
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("icon: không tạo được thư mục ảnh: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".icon-*.png")
	if err != nil {
		return fmt.Errorf("icon: không tạo được file tạm: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := png.Encode(tmp, img); err != nil {
		tmp.Close()
		return fmt.Errorf("icon: không ghi được ảnh PNG: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("icon: không ghi được ảnh PNG: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("icon: không đóng được file ảnh: %w", err)
	}
	if err := os.Rename(tmpName, filepath.Join(dir, name)); err != nil {
		return fmt.Errorf("icon: không thay thế được file ảnh: %w", err)
	}
	return nil
}

// ValidName reports whether a file name is a plain name inside the icons
// directory. Anything with a path separator or a parent reference is rejected
// so a crafted name cannot read files outside the icons folder.
func ValidName(name string) bool {
	if name == "" {
		return false
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return false
	}
	if filepath.Base(name) != name {
		return false
	}
	return true
}

// ReadDataURI reads a PNG from the icons directory and returns it as a data
// URI so the frontend can show it without an HTTP request.
func ReadDataURI(dir, name string) (string, error) {
	if !ValidName(name) {
		return "", ErrBadName
	}
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return "", fmt.Errorf("icon: không đọc được ảnh %s: %w", name, err)
	}
	return DataURI(data), nil
}

// DataURI encodes PNG bytes as an inline data URI.
func DataURI(pngBytes []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
}

// EncodePNG encodes an image as PNG bytes.
func EncodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("icon: không mã hoá được ảnh PNG: %w", err)
	}
	return buf.Bytes(), nil
}

// FileSize returns the size of a file in the icons directory, or 0 when it
// does not exist.
func FileSize(dir, name string) int {
	if !ValidName(name) {
		return 0
	}
	info, err := os.Stat(filepath.Join(dir, name))
	if err != nil {
		return 0
	}
	return int(info.Size())
}

// Delete removes an icon file. A missing file is not an error.
func Delete(dir, name string) {
	if !ValidName(name) {
		return
	}
	_ = os.Remove(filepath.Join(dir, name))
}
