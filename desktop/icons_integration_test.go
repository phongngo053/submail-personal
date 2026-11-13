package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phongngo053/submail-personal/internal/integrations"
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
