package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/phongngo053/submail-personal/internal/icon"
	"github.com/phongngo053/submail-personal/internal/integrations"
)

// iconSize is the edge length of a subscription or avatar icon in pixels.
const iconSize = 64

// appIconSize is the edge length of the full resolution application icon.
const appIconSize = 1024

// IconResult carries a freshly generated icon back to the frontend.
type IconResult struct {
	// File is the name of the icon inside the icons directory.
	File string `json:"file"`
	// DataURI is the PNG as an inline data URI so the UI can show it right
	// away without another round trip.
	DataURI string `json:"dataUri"`
	// Bytes is the size of the stored PNG file.
	Bytes int `json:"bytes"`
}

// GetIconPresets returns the built in catalogue of popular subscriptions.
func (a *App) GetIconPresets() []icon.Preset {
	return icon.Presets()
}

// GetIconData returns an icon as a data URI. The frontend asks for an icon
// only once and then keeps it in its own cache.
func (a *App) GetIconData(file string) (string, error) {
	if a.store == nil {
		return "", fmt.Errorf("%w: ứng dụng chưa sẵn sàng", ErrInvalidInput)
	}
	return icon.ReadDataURI(a.store.IconsDir(), file)
}

// GenerateSubscriptionIcon creates an icon for one subscription with the AI
// image service and attaches it to the subscription.
//
// The image request takes 15 to 30 seconds, so it runs outside the state lock:
// the rest of the app stays usable while the picture is being made.
func (a *App) GenerateSubscriptionIcon(id int64) (IconResult, error) {
	a.mu.RLock()
	var name, category string
	for _, sub := range a.state.Subscriptions {
		if sub.ID == id {
			name = sub.Name
			category = sub.Category
			break
		}
	}
	a.mu.RUnlock()

	if name == "" {
		return IconResult{}, ErrNotFound
	}

	prompt := icon.BuildSubscriptionPrompt(name, category)
	result, err := a.generateIcon(prompt, fmt.Sprintf("sub-%d.png", id), iconSize)
	if err != nil {
		return IconResult{}, err
	}
	if _, err := a.AttachSubscriptionIcon(id, result.File); err != nil {
		return IconResult{}, err
	}
	return result, nil
}

// GenerateAccountAvatar creates an avatar for one Gmail account with the AI
// image service.
func (a *App) GenerateAccountAvatar(email string) (IconResult, error) {
	a.mu.RLock()
	var tag, color string
	for _, acc := range a.state.GmailAccounts {
		if acc.Email == email {
			tag = acc.Tag
			color = acc.Color
			break
		}
	}
	a.mu.RUnlock()

	if tag == "" && email == "" {
		return IconResult{}, ErrNotFound
	}

	prompt := icon.BuildAvatarPrompt(tag, color)
	result, err := a.generateIcon(prompt, accountIconName(email), iconSize)
	if err != nil {
		return IconResult{}, err
	}
	if _, err := a.AttachAccountIcon(email, result.File); err != nil {
		return IconResult{}, err
	}
	return result, nil
}

// GenerateAppIcon creates the artwork used for the desktop application icon.
func (a *App) GenerateAppIcon() (IconResult, error) {
	return a.generateIcon(icon.BuildAppPrompt(), "app-icon.png", appIconSize)
}

// generateIcon asks the image service for a picture, stores it in the icons
// directory and returns the result. The file name must be a plain name.
//
// The image service is an optional integration: it is configured in a local
// file that never leaves the machine. When no integration is enabled the call
// is refused and the UI hides the action.
func (a *App) generateIcon(prompt, fileName string, size int) (IconResult, error) {
	if a.store == nil {
		return IconResult{}, fmt.Errorf("%w: ứng dụng chưa sẵn sàng", ErrInvalidInput)
	}
	if !icon.ValidName(fileName) {
		return IconResult{}, icon.ErrBadName
	}

	reg, err := integrations.LoadRegistry()
	if err != nil {
		return IconResult{}, err
	}
	it, ok := reg.FirstEnabled(integrations.TypeImage)
	if !ok {
		return IconResult{}, fmt.Errorf(
			"%w: chưa cấu hình dịch vụ tạo ảnh trong %s",
			integrations.ErrUnsupported, integrations.ConfigEnv)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	client := integrations.NewImageClient(it)
	data, err := client.Generate(ctx, prompt, "")
	if err != nil {
		return IconResult{}, err
	}

	img, err := icon.DecodePNG(data)
	if err != nil {
		return IconResult{}, err
	}

	resized := icon.Resize(img, size)
	if err := a.store.EnsureIconsDir(); err != nil {
		return IconResult{}, fmt.Errorf("%w: không tạo được thư mục ảnh: %v", ErrInvalidInput, err)
	}
	dir := a.store.IconsDir()
	if err := icon.SavePNG(dir, fileName, resized); err != nil {
		return IconResult{}, err
	}

	return IconResult{
		File:    fileName,
		DataURI: icon.DataURI(mustEncodePNG(resized)),
		Bytes:   icon.FileSize(dir, fileName),
	}, nil
}

// AttachSubscriptionIcon stores the icon file name on a subscription. It is
// separate from the generation so the frontend can attach an icon it already
// holds without another round trip to the image service.
func (a *App) AttachSubscriptionIcon(id int64, file string) (Snapshot, error) {
	if !icon.ValidName(file) {
		return Snapshot{}, icon.ErrBadName
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	for i := range a.state.Subscriptions {
		if a.state.Subscriptions[i].ID != id {
			continue
		}
		a.state.Subscriptions[i].IconFile = file
		if err := a.persist(); err != nil {
			return Snapshot{}, err
		}
		return a.snapshotLocked(), nil
	}
	return Snapshot{}, ErrNotFound
}

// AttachAccountIcon stores the icon file name on a Gmail account.
func (a *App) AttachAccountIcon(email, file string) (Snapshot, error) {
	if !icon.ValidName(file) {
		return Snapshot{}, icon.ErrBadName
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	for i := range a.state.GmailAccounts {
		if a.state.GmailAccounts[i].Email != email {
			continue
		}
		a.state.GmailAccounts[i].IconFile = file
		if err := a.persist(); err != nil {
			return Snapshot{}, err
		}
		return a.snapshotLocked(), nil
	}
	return Snapshot{}, ErrNotFound
}

// accountIconName derives a stable file name from an email address so the
// same account always reuses the same icon file.
func accountIconName(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return "acc-" + hex.EncodeToString(sum[:])[:12] + ".png"
}

// mustEncodePNG encodes an image as PNG bytes. Encoding a 64 or 1024 pixel
// image cannot fail, so a failure here would be a programming error.
func mustEncodePNG(img image.Image) []byte {
	data, err := icon.EncodePNG(img)
	if err != nil {
		return nil
	}
	return data
}
