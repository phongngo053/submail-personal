package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phongngo053/submail-personal/internal/integrations"
	"github.com/phongngo053/submail-personal/internal/model"
)

// writeImageIntegration points the app at a fake image service and returns
// the PNG bytes it hands out.
func writeImageIntegration(t *testing.T, png []byte) {
	t.Helper()

	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(png)
	}))
	t.Cleanup(cdn.Close)

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(integrations.ImageResponse{Data: []integrations.ImageData{
			{URL: cdn.URL + "/image.png"},
		}})
	}))
	t.Cleanup(api.Close)

	path := filepath.Join(t.TempDir(), "integrations.json")
	body := `{"integrations":[{"id":"img","name":"Test","type":"image","baseUrl":"` + api.URL + `","enabled":true}]}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(integrations.ConfigEnv, path)
}

// testPNG returns a tiny valid PNG.
func testPNG(t *testing.T) []byte {
	t.Helper()
	return solidTestPNG()
}

// solidTestPNG encodes a small solid image as PNG bytes.
func solidTestPNG() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, color.NRGBA{R: 70, G: 95, B: 255, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// TestGenerateSubscriptionIconStoresTheFile checks a generated icon lands on
// disk and is attached to the subscription.
func TestGenerateSubscriptionIconStoresTheFile(t *testing.T) {
	app := newTestApp(t)
	writeImageIntegration(t, testPNG(t))

	id := app.state.Subscriptions[0].ID
	result, err := app.GenerateSubscriptionIcon(id)
	if err != nil {
		t.Fatalf("GenerateSubscriptionIcon() error = %v", err)
	}
	if result.File != "sub-1.png" {
		t.Errorf("file = %q, want %q", result.File, "sub-1.png")
	}
	if !strings.HasPrefix(result.DataURI, "data:image/png;base64,") {
		t.Errorf("data URI = %q, want a PNG data URI", result.DataURI)
	}
	if result.Bytes <= 0 {
		t.Error("the icon file is empty")
	}

	if _, err := os.Stat(filepath.Join(app.store.IconsDir(), result.File)); err != nil {
		t.Fatalf("icon file missing on disk: %v", err)
	}

	snap := app.GetState()
	for _, sub := range snap.Subscriptions {
		if sub.ID == id && sub.IconFile != result.File {
			t.Errorf("iconFile = %q, want %q", sub.IconFile, result.File)
		}
	}
}

// TestGetIconDataReturnsTheIcon checks the frontend can fetch a stored icon.
func TestGetIconDataReturnsTheIcon(t *testing.T) {
	app := newTestApp(t)
	writeImageIntegration(t, testPNG(t))

	result, err := app.GenerateSubscriptionIcon(app.state.Subscriptions[0].ID)
	if err != nil {
		t.Fatalf("GenerateSubscriptionIcon() error = %v", err)
	}

	dataURI, err := app.GetIconData(result.File)
	if err != nil {
		t.Fatalf("GetIconData() error = %v", err)
	}
	if !strings.HasPrefix(dataURI, "data:image/png;base64,") {
		t.Errorf("data URI = %q, want a PNG data URI", dataURI)
	}

	// The payload must be real base64 that decodes to a PNG.
	payload := strings.TrimPrefix(dataURI, "data:image/png;base64,")
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("the data URI does not hold valid base64: %v", err)
	}
	if len(decoded) == 0 {
		t.Error("the data URI holds no image data")
	}
}

// TestGetIconDataRefusesTraversal checks a crafted file name cannot read a
// file outside the icons folder.
func TestGetIconDataRefusesTraversal(t *testing.T) {
	app := newTestApp(t)

	for _, name := range []string{"../data.json", "..\\data.json", "/etc/passwd", "sub/../../data.json"} {
		if _, err := app.GetIconData(name); err == nil {
			t.Errorf("GetIconData(%q) succeeded, want an error", name)
		}
	}
}

// TestGenerateIconWithoutIntegrationIsRefused checks the action reports a
// clear error when no image service is configured.
func TestGenerateIconWithoutIntegrationIsRefused(t *testing.T) {
	app := newTestApp(t)
	t.Setenv(integrations.ConfigEnv, filepath.Join(t.TempDir(), "missing.json"))

	if _, err := app.GenerateSubscriptionIcon(app.state.Subscriptions[0].ID); err == nil {
		t.Fatal("GenerateSubscriptionIcon() expected error when no integration is configured")
	}
}

// TestGenerateIconForUnknownSubscription checks a missing subscription is
// reported instead of creating a stray file.
func TestGenerateIconForUnknownSubscription(t *testing.T) {
	app := newTestApp(t)
	writeImageIntegration(t, testPNG(t))

	if _, err := app.GenerateSubscriptionIcon(9999); err == nil {
		t.Fatal("GenerateSubscriptionIcon() expected error for an unknown id")
	}
}

// TestGenerateAccountAvatarStoresTheFile checks the avatar is created and
// attached to the account.
func TestGenerateAccountAvatarStoresTheFile(t *testing.T) {
	app := newTestApp(t)
	writeImageIntegration(t, testPNG(t))

	email := app.state.GmailAccounts[0].Email
	result, err := app.GenerateAccountAvatar(email)
	if err != nil {
		t.Fatalf("GenerateAccountAvatar() error = %v", err)
	}
	if !strings.HasPrefix(result.File, "acc-") {
		t.Errorf("file = %q, want an account icon name", result.File)
	}

	snap := app.GetState()
	for _, acc := range snap.GmailAccounts {
		if acc.Email == email && acc.IconFile != result.File {
			t.Errorf("iconFile = %q, want %q", acc.IconFile, result.File)
		}
	}
}

// TestAccountIconNameIsStable checks the same account always reuses the same
// icon file.
func TestAccountIconNameIsStable(t *testing.T) {
	first := accountIconName("User@Example.com")
	second := accountIconName("user@example.com")

	if first != second {
		t.Errorf("names differ: %q and %q", first, second)
	}
	if !strings.HasSuffix(first, ".png") {
		t.Errorf("name = %q, want a .png file name", first)
	}
}

// TestGetIconPresets returns the catalogue the picker shows.
func TestGetIconPresets(t *testing.T) {
	app := newTestApp(t)

	presets := app.GetIconPresets()
	if len(presets) == 0 {
		t.Fatal("no presets were returned")
	}
	if presets[0].Name == "" || presets[0].Prompt == "" {
		t.Errorf("the first preset is incomplete: %+v", presets[0])
	}
}

// TestSaveSubscriptionKeepsCustomPassWhileLocked checks the secret of a
// subscription survives a save that happens while the vault is locked.
//
// The frontend only ever holds the masked value in that case, so writing it
// back would destroy the stored password.
func TestSaveSubscriptionKeepsCustomPassWhileLocked(t *testing.T) {
	app := newTestApp(t)
	app.locked = true

	id := app.state.Subscriptions[1].ID
	if app.state.Subscriptions[1].CustomPass == "" {
		t.Fatal("the fixture subscription has no custom password")
	}

	snap, err := app.SaveSubscription(SubscriptionView{
		ID:         id,
		Name:       "Dịch vụ riêng",
		Category:   "Media",
		Gmail:      fixtureEmail,
		Price:      10,
		Cycle:      "tháng",
		Status:     model.StatusActive,
		AuthMode:   model.AuthCustom,
		CustomPass: "••••••••",
	})
	if err != nil {
		t.Fatalf("SaveSubscription() error = %v", err)
	}

	for _, sub := range snap.Subscriptions {
		if sub.ID != id {
			continue
		}
		if sub.CustomPass != "" {
			t.Errorf("customPass = %q, want the masked value while locked", sub.CustomPass)
		}
	}

	// The stored secret must still be the real one.
	if app.state.Subscriptions[1].CustomPass != "fixture-custom-pass" {
		t.Errorf("stored customPass = %q, want it to be kept", app.state.Subscriptions[1].CustomPass)
	}
}

// TestSaveGmailKeepsPasswordWhileLocked checks the same for a Gmail account.
func TestSaveGmailKeepsPasswordWhileLocked(t *testing.T) {
	app := newTestApp(t)
	app.locked = true

	if _, err := app.SaveGmail(AccountView{
		Email:      fixtureEmail,
		Tag:        "Mail Chính",
		Color:      "blue",
		Password:   "••••••••••••••",
		Secret2FA:  "••••••••••••",
		AppPassword: "••••••••••••••",
	}); err != nil {
		t.Fatalf("SaveGmail() error = %v", err)
	}

	if app.state.GmailAccounts[0].Password != fixturePassword {
		t.Errorf("password = %q, want it to be kept", app.state.GmailAccounts[0].Password)
	}
	if app.state.GmailAccounts[0].Secret2FA != fixtureSecret {
		t.Errorf("secret = %q, want it to be kept", app.state.GmailAccounts[0].Secret2FA)
	}
	if app.state.GmailAccounts[0].AppPassword != fixtureAppPass {
		t.Errorf("app password = %q, want it to be kept", app.state.GmailAccounts[0].AppPassword)
	}
}

// TestSaveGmailUpdatesPasswordWhileUnlocked checks the normal path still
// writes the real password.
func TestSaveGmailUpdatesPasswordWhileUnlocked(t *testing.T) {
	app := newTestApp(t)
	app.locked = false

	if _, err := app.SaveGmail(AccountView{
		Email:    fixtureEmail,
		Tag:      "Mail Chính",
		Color:    "blue",
		Password: "a-brand-new-password",
	}); err != nil {
		t.Fatalf("SaveGmail() error = %v", err)
	}

	if app.state.GmailAccounts[0].Password != "a-brand-new-password" {
		t.Errorf("password = %q, want the new value", app.state.GmailAccounts[0].Password)
	}
}
