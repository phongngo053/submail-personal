// Package integrations connects SubMail Personal to optional services that run
// on the user's own machine.
//
// The app ships with no integrations enabled. A user who wants AI generated
// icons points the app at their own image service by dropping a configuration
// file next to the data file:
//
//	%AppData%\SubMailPersonal\integrations.json
//
// That file is never committed to the repository and never leaves the
// machine: the app only reports the id, name, type and enabled flag of each
// integration to the frontend. The base URL and the API key stay in Go.
package integrations

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigEnv names the environment variable that points at the configuration
// file. It lets a user keep the file outside the app data directory.
const ConfigEnv = "SUBMAIL_INTEGRATIONS_CONFIG"

// Integration types. The type decides which adapter handles the integration.
const (
	// TypeImage is an OpenAI-compatible text-to-image service.
	TypeImage = "image"
)

// Errors reported to the user. The frontend maps them to Vietnamese toasts.
var (
	ErrNotFound    = errors.New("Không tìm thấy tích hợp")
	ErrDisabled    = errors.New("Tích hợp đang tắt")
	ErrBadConfig   = errors.New("File cấu hình tích hợp không hợp lệ")
	ErrUnsupported = errors.New("Loại tích hợp không được hỗ trợ")
)

// Integration is one configured service.
type Integration struct {
	// ID is the stable identifier used by the frontend to call the adapter.
	ID string `json:"id"`
	// Name is the label shown in the UI.
	Name string `json:"name"`
	// Type selects the adapter, currently only "image".
	Type string `json:"type"`
	// BaseURL is the root URL of the service, without a trailing slash.
	BaseURL string `json:"baseUrl"`
	// APIKey is the bearer token sent with every request. It never leaves Go.
	APIKey string `json:"apiKey"`
	// Enabled reports whether the user wants this integration used.
	Enabled bool `json:"enabled"`
	// Model is the optional model name passed to the image service.
	Model string `json:"model,omitempty"`
}

// Config is the on-disk configuration document.
type Config struct {
	Integrations []Integration `json:"integrations"`
}

// View is the public description of an integration. It carries no secret, so
// it is safe to hand to the frontend.
type View struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
}

// DefaultPath returns the configuration file path inside the OS app-data dir.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("integrations: cannot resolve app data dir: %w", err)
	}
	return filepath.Join(dir, "SubMailPersonal", "integrations.json"), nil
}

// ConfigPath returns the file the configuration is read from. The environment
// variable wins so a user can keep the file anywhere.
func ConfigPath() (string, error) {
	if custom := strings.TrimSpace(os.Getenv(ConfigEnv)); custom != "" {
		return custom, nil
	}
	return DefaultPath()
}

// Load reads the configuration. A missing file is not an error: the app simply
// has no integrations. A malformed file is reported so the user can fix it.
func Load() (Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("integrations: cannot read %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("%w: %s", ErrBadConfig, err)
	}
	for i := range cfg.Integrations {
		it := &cfg.Integrations[i]
		it.ID = strings.TrimSpace(it.ID)
		it.Type = strings.TrimSpace(it.Type)
		it.BaseURL = strings.TrimRight(strings.TrimSpace(it.BaseURL), "/")
		if it.ID == "" || it.Type == "" || it.BaseURL == "" {
			return Config{}, fmt.Errorf("%w: mỗi tích hợp cần id, type và baseUrl", ErrBadConfig)
		}
	}
	return cfg, nil
}

// Registry holds the configured integrations and hands out their public
// descriptions. It is safe for concurrent use.
type Registry struct {
	integrations map[string]Integration
}

// NewRegistry builds a registry from a configuration.
func NewRegistry(cfg Config) *Registry {
	reg := &Registry{integrations: make(map[string]Integration, len(cfg.Integrations))}
	for _, it := range cfg.Integrations {
		reg.integrations[it.ID] = it
	}
	return reg
}

// LoadRegistry reads the configuration from disk and builds a registry.
func LoadRegistry() (*Registry, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}
	return NewRegistry(cfg), nil
}

// Views returns the public descriptions of every configured integration.
func (r *Registry) Views() []View {
	views := make([]View, 0, len(r.integrations))
	for _, it := range r.integrations {
		views = append(views, View{ID: it.ID, Name: it.Name, Type: it.Type, Enabled: it.Enabled})
	}
	return views
}

// Get returns the full configuration of one integration. The API key is
// included, so callers must never pass the result to the frontend.
func (r *Registry) Get(id string) (Integration, error) {
	it, ok := r.integrations[id]
	if !ok {
		return Integration{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	return it, nil
}

// Enabled returns the configuration of one integration and refuses when the
// user has switched it off.
func (r *Registry) Enabled(id string) (Integration, error) {
	it, err := r.Get(id)
	if err != nil {
		return Integration{}, err
	}
	if !it.Enabled {
		return Integration{}, fmt.Errorf("%w: %s", ErrDisabled, id)
	}
	return it, nil
}

// FirstEnabled returns the first enabled integration of the given type. The
// frontend uses it to decide whether to show the matching action at all.
func (r *Registry) FirstEnabled(integrationType string) (Integration, bool) {
	for _, it := range r.integrations {
		if it.Enabled && it.Type == integrationType {
			return it, true
		}
	}
	return Integration{}, false
}
