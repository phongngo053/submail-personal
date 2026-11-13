// Command genicon creates the artwork used for the desktop application icon.
//
// It calls the text-to-image service configured in the local integrations
// file, writes a 1024x1024 PNG next to the build files and packs it into a
// multi resolution Windows .ico file.
//
//	go run ./cmd/genicon
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/phongngo053/submail-personal/internal/icon"
	"github.com/phongngo053/submail-personal/internal/integrations"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Lỗi:", err)
		os.Exit(1)
	}
}

func run() error {
	root, err := projectRoot()
	if err != nil {
		return err
	}

	reg, err := integrations.LoadRegistry()
	if err != nil {
		return err
	}
	it, ok := reg.FirstEnabled(integrations.TypeImage)
	if !ok {
		return fmt.Errorf(
			"chưa cấu hình dịch vụ tạo ảnh trong %s", integrations.ConfigEnv)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	fmt.Println("Đang tạo ảnh icon ứng dụng…")
	client := integrations.NewImageClient(it)
	data, err := client.Generate(ctx, icon.BuildAppPrompt(), "")
	if err != nil {
		return err
	}

	img, err := icon.DecodePNG(data)
	if err != nil {
		return err
	}

	// The PNG keeps the full resolution so it can be used as the window icon.
	pngPath := filepath.Join(root, "build", "appicon.png")
	if err := os.MkdirAll(filepath.Dir(pngPath), 0o755); err != nil {
		return fmt.Errorf("không tạo được thư mục build: %w", err)
	}
	if err := os.WriteFile(pngPath, data, 0o644); err != nil {
		return fmt.Errorf("không ghi được %s: %w", pngPath, err)
	}
	fmt.Println("Đã ghi", pngPath)

	icoPath := filepath.Join(root, "build", "windows", "icon.ico")
	if err := os.MkdirAll(filepath.Dir(icoPath), 0o755); err != nil {
		return fmt.Errorf("không tạo được thư mục build/windows: %w", err)
	}

	file, err := os.Create(icoPath)
	if err != nil {
		return fmt.Errorf("không tạo được %s: %w", icoPath, err)
	}
	defer file.Close()

	if err := icon.WriteICO(file, img, nil); err != nil {
		return err
	}
	fmt.Println("Đã ghi", icoPath)

	return nil
}

// projectRoot returns the desktop folder, which holds the build directory.
//
// The command is meant to be started from the desktop folder, so the folder
// that contains cmd/genicon is the one to use. When it is started from
// somewhere else the current folder is used instead, as long as it looks like
// the project root.
func projectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("không xác định được thư mục làm việc: %w", err)
	}
	dir = filepath.Clean(dir)

	// Walk up from the current folder until the desktop project root shows up.
	for i := 0; i < 4; i++ {
		if _, err := os.Stat(filepath.Join(dir, "wails.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("không tìm thấy thư mục gốc dự án (cần file wails.json)")
}
