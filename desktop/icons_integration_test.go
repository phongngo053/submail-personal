package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phongngo053/submail-personal/internal/integrations"
	"github.com/phongngo053/submail-personal/internal/model"
)

// TestImageIntegrationAgainstRealService exercises the whole icon pipeline
// against the image service the user configured on this machine.
//
// The test is skipped unless SUBMAIL_IMAGE_INTEGRATION is set, because it
// calls a real service and takes 15 to 30 seconds.
func TestImageIntegrationAgainstRealService(t *testing.T) {
	if os.Getenv("SUBMAIL_IMAGE_INTEGRATION") == "" {
		t.Skip("set SUBMAIL_IMAGE_INTEGRATION=1 to run the test against the real image service")
	}

	app := newTestApp(t)

	result, err := app.GenerateSubscriptionIcon(app.state.Subscriptions[0].ID)
	if err != nil {
		t.Fatalf("GenerateSubscriptionIcon() error = %v", err)
	}
	if result.File == "" || result.Bytes <= 0 {
		t.Fatalf("the generated icon is empty: %+v", result)
	}
	if !strings.HasPrefix(result.DataURI, "data:image/png;base64,") {
		t.Errorf("data URI = %q, want a PNG data URI", result.DataURI[:32])
	}

	if _, err := os.Stat(filepath.Join(app.store.IconsDir(), result.File)); err != nil {
		t.Fatalf("icon file missing on disk: %v", err)
	}

	// The icon must also be attached to the subscription.
	snap := app.GetState()
	found := false
	for _, sub := range snap.Subscriptions {
		if sub.IconFile == result.File {
			found = true
		}
	}
	if !found {
		t.Error("the generated icon was not attached to the subscription")
	}

	t.Logf("generated %s (%d bytes)", result.File, result.Bytes)
}

// TestImageIntegrationConfigIsReadable checks the configuration file the user
// wrote can be loaded and describes an enabled image service.
func TestImageIntegrationConfigIsReadable(t *testing.T) {
	if os.Getenv("SUBMAIL_IMAGE_INTEGRATION") == "" {
		t.Skip("set SUBMAIL_IMAGE_INTEGRATION=1 to run the test against the real image service")
	}

	reg, err := integrations.LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}
	it, ok := reg.FirstEnabled(integrations.TypeImage)
	if !ok {
		t.Fatal("no enabled image integration was found")
	}
	t.Logf("using image service %q at %s", it.ID, it.BaseURL)
}

// TestFrontendIconFlow walks the exact sequence the UI performs: save a
// subscription, generate its icon, then read the picture back for display.
func TestFrontendIconFlow(t *testing.T) {
	if os.Getenv("SUBMAIL_IMAGE_INTEGRATION") == "" {
		t.Skip("set SUBMAIL_IMAGE_INTEGRATION=1 to run the test against the real image service")
	}

	app := newTestApp(t)

	// 1. The user fills in the form and saves it.
	snap, err := app.SaveSubscription(SubscriptionView{
		Name:     "Netflix Premium 4K",
		Plan:     "Gói Ultra HD 4 thiết bị",
		Category: "Media",
		Gmail:    app.state.GmailAccounts[0].Email,
		Price:    10.50,
		Cycle:    "tháng",
		NextDate: "2026-12-01",
		Payment:  "Visa",
		Status:   model.StatusActive,
		AuthMode: model.AuthSSO,
		IconBg:   "bg-red-700",
		IconText: "N",
	})
	if err != nil {
		t.Fatalf("SaveSubscription() error = %v", err)
	}

	// The new subscription sits at the top of the list, exactly like the UI
	// expects when it picks the record it just created.
	if len(snap.Subscriptions) == 0 {
		t.Fatal("the saved subscription is missing")
	}
	created := snap.Subscriptions[0]
	if created.Name != "Netflix Premium 4K" {
		t.Fatalf("name = %q, want the saved name", created.Name)
	}

	// 2. The user asks for an icon.
	result, err := app.GenerateSubscriptionIcon(created.ID)
	if err != nil {
		t.Fatalf("GenerateSubscriptionIcon() error = %v", err)
	}

	// 3. The frontend fetches the picture to show it.
	dataURI, err := app.GetIconData(result.File)
	if err != nil {
		t.Fatalf("GetIconData() error = %v", err)
	}
	if len(dataURI) < 100 {
		t.Errorf("data URI is only %d characters, want a real image", len(dataURI))
	}

	// 4. The snapshot carries the file name so the row can show the picture.
	snap = app.GetState()
	found := false
	for _, sub := range snap.Subscriptions {
		if sub.ID == created.ID {
			found = true
			if sub.IconFile != result.File {
				t.Errorf("iconFile = %q, want %q", sub.IconFile, result.File)
			}
		}
	}
	if !found {
		t.Error("the subscription disappeared from the snapshot")
	}

	t.Logf("icon %s (%d bytes) is ready to display", result.File, result.Bytes)
}
