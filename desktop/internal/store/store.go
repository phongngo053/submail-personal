// Package store loads and saves the SubMail Personal state as a JSON document
// on disk, falling back to demo data when the file is missing or corrupt.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/phongngo053/submail-personal/internal/model"
)

// Store reads and writes the state file.
type Store struct {
	path string
}

// New returns a Store backed by the given file path.
func New(path string) *Store {
	return &Store{path: path}
}

// DefaultPath returns the per-user data file path inside the OS app-data dir.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("store: cannot resolve app data dir: %w", err)
	}
	return filepath.Join(dir, "SubMailPersonal", "data.json"), nil
}

// Path returns the file path this store writes to.
func (s *Store) Path() string { return s.path }

// IconsDir returns the directory holding AI-generated icons. It sits next to
// the data file so the whole app state stays in one folder.
func (s *Store) IconsDir() string {
	return filepath.Join(filepath.Dir(s.path), "icons")
}

// EnsureIconsDir creates the icons directory if it does not exist yet.
func (s *Store) EnsureIconsDir() error {
	return os.MkdirAll(s.IconsDir(), 0o700)
}

// Load reads the state file. A missing or corrupt file yields an empty state
// instead of an error, so the app always starts. The user then adds their own
// accounts; the app never ships with pre-filled credentials.
func (s *Store) Load() model.State {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return model.NewState()
	}

	var st model.State
	if err := json.Unmarshal(data, &st); err != nil {
		return model.NewState()
	}
	st.Normalize()
	return st
}

// Save writes the state atomically: the document goes to a temporary file in
// the same directory and is then renamed over the target, so a crash mid-write
// cannot leave a half-written file.
func (s *Store) Save(st model.State) error {
	st.Normalize()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("store: cannot create data dir: %w", err)
	}

	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("store: cannot encode state: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".data-*.json")
	if err != nil {
		return fmt.Errorf("store: cannot create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("store: cannot write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("store: cannot sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("store: cannot close temp file: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("store: cannot replace data file: %w", err)
	}
	return nil
}

// Export writes the state to an arbitrary path (used for JSON backups).
func Export(st model.State, path string) error {
	st.Normalize()
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("store: cannot encode state: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("store: cannot create export dir: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}

// Import reads a state document from an arbitrary path.
func Import(path string) (model.State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.State{}, fmt.Errorf("store: cannot read %s: %w", path, err)
	}
	var st model.State
	if err := json.Unmarshal(data, &st); err != nil {
		return model.State{}, fmt.Errorf("store: %s is not a valid SubMail backup: %w", path, err)
	}
	if st.GmailAccounts == nil || st.Subscriptions == nil {
		return model.State{}, fmt.Errorf("store: %s is missing gmailAccounts or subscriptionsData", path)
	}
	st.Normalize()
	return st, nil
}

// EmptyState returns a state with no accounts and no subscriptions. The app
// ships without any pre-filled data: every credential in the vault belongs to
// the person running it.
func EmptyState() model.State {
	return model.NewState()
}
