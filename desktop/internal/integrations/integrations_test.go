package integrations

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeConfig writes a configuration file and points the environment variable
// at it for the duration of the test.
func writeConfig(t *testing.T, body string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "integrations.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(ConfigEnv, path)
}

// TestLoadMissingFileIsNotAnError checks the app starts cleanly when the user
// never configured an integration.
func TestLoadMissingFileIsNotAnError(t *testing.T) {
	t.Setenv(ConfigEnv, filepath.Join(t.TempDir(), "nope.json"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Integrations) != 0 {
		t.Errorf("got %d integrations, want 0", len(cfg.Integrations))
	}
}

// TestLoadRejectsBrokenConfig checks a malformed file is reported instead of
// silently ignored.
func TestLoadRejectsBrokenConfig(t *testing.T) {
	writeConfig(t, "{not json")

	if _, err := Load(); err == nil {
		t.Fatal("Load() expected error for a malformed config file")
	}
}

// TestLoadRejectsIncompleteIntegration checks that an entry missing id, type
// or baseUrl is refused.
func TestLoadRejectsIncompleteIntegration(t *testing.T) {
	writeConfig(t, `{"integrations":[{"id":"img","type":"image"}]}`)

	if _, err := Load(); err == nil {
		t.Fatal("Load() expected error for an incomplete integration")
	}
}

// TestLoadTrimsBaseURL checks the trailing slash is removed so the adapter
// can join paths safely.
func TestLoadTrimsBaseURL(t *testing.T) {
	writeConfig(t, `{"integrations":[{"id":"img","type":"image","baseUrl":"http://localhost:9000/","enabled":true}]}`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Integrations[0].BaseURL != "http://localhost:9000" {
		t.Errorf("baseUrl = %q, want the trimmed value", cfg.Integrations[0].BaseURL)
	}
}

// TestViewsNeverCarryTheAPIKey checks the public description is safe to send
// to the frontend.
func TestViewsNeverCarryTheAPIKey(t *testing.T) {
	cfg := Config{Integrations: []Integration{
		{ID: "img", Name: "Image API", Type: TypeImage, BaseURL: "http://localhost:9000", APIKey: "super-secret", Enabled: true},
	}}
	reg := NewRegistry(cfg)

	views := reg.Views()
	if len(views) != 1 {
		t.Fatalf("got %d views, want 1", len(views))
	}
	if views[0].ID != "img" || views[0].Type != TypeImage || !views[0].Enabled {
		t.Errorf("unexpected view: %+v", views[0])
	}

	encoded, err := json.Marshal(views)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "super-secret") {
		t.Error("the API key leaked into the public view")
	}
}

// TestEnabledRefusesDisabledIntegration checks a switched-off integration
// cannot be used.
func TestEnabledRefusesDisabledIntegration(t *testing.T) {
	reg := NewRegistry(Config{Integrations: []Integration{
		{ID: "img", Type: TypeImage, BaseURL: "http://localhost:9000", Enabled: false},
	}})

	if _, err := reg.Enabled("img"); err == nil {
		t.Fatal("Enabled() expected error for a disabled integration")
	}
	if _, err := reg.Enabled("missing"); err == nil {
		t.Fatal("Enabled() expected error for an unknown integration")
	}
}

// TestFirstEnabled checks the frontend can tell whether an action is
// available at all.
func TestFirstEnabled(t *testing.T) {
	reg := NewRegistry(Config{Integrations: []Integration{
		{ID: "off", Type: TypeImage, BaseURL: "http://localhost:9000", Enabled: false},
		{ID: "on", Type: TypeImage, BaseURL: "http://localhost:9001", Enabled: true},
	}})

	it, ok := reg.FirstEnabled(TypeImage)
	if !ok {
		t.Fatal("FirstEnabled() found no enabled image integration")
	}
	if it.ID != "on" {
		t.Errorf("id = %q, want %q", it.ID, "on")
	}

	if _, ok := reg.FirstEnabled("video"); ok {
		t.Error("FirstEnabled() found an integration of an unknown type")
	}
}

// TestImageClientReturnsInlineBase64 checks the adapter accepts the inline
// base64 answer.
func TestImageClientReturnsInlineBase64(t *testing.T) {
	want := []byte("fake-png-bytes")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			t.Errorf("path = %q, want /v1/images/generations", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("authorization = %q, want the bearer token", got)
		}
		json.NewEncoder(w).Encode(ImageResponse{Data: []ImageData{
			{B64JSON: base64.StdEncoding.EncodeToString(want)},
		}})
	}))
	defer server.Close()

	client := NewImageClient(Integration{BaseURL: server.URL, APIKey: "test-key"})
	got, err := client.Generate(context.Background(), "a cat", "")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("image = %q, want %q", got, want)
	}
}

// TestImageClientDownloadsRemoteURL checks the adapter follows a remote URL
// when the service answers with one.
func TestImageClientDownloadsRemoteURL(t *testing.T) {
	want := []byte("remote-png-bytes")
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(want)
	}))
	defer cdn.Close()

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ImageResponse{Data: []ImageData{
			{URL: cdn.URL + "/image.png"},
		}})
	}))
	defer api.Close()

	client := NewImageClient(Integration{BaseURL: api.URL})
	got, err := client.Generate(context.Background(), "a dog", "")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("image = %q, want %q", got, want)
	}
}

// TestImageClientReportsServiceError checks a failed call surfaces the
// service message instead of an empty image.
func TestImageClientReportsServiceError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	defer server.Close()

	client := NewImageClient(Integration{BaseURL: server.URL, APIKey: "wrong"})
	_, err := client.Generate(context.Background(), "a cat", "")
	if err == nil {
		t.Fatal("Generate() expected error for a 401 answer")
	}
	if !strings.Contains(err.Error(), "invalid api key") {
		t.Errorf("error = %v, want it to mention the service message", err)
	}
}

// TestImageClientRejectsEmptyPrompt checks the prompt is validated before any
// request is made.
func TestImageClientRejectsEmptyPrompt(t *testing.T) {
	client := NewImageClient(Integration{BaseURL: "http://localhost:9"})

	if _, err := client.Generate(context.Background(), "   ", ""); err == nil {
		t.Fatal("Generate() expected error for an empty prompt")
	}
}
