package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phongngo053/submail-personal/internal/model"
)

// fixtureState returns a small state used across the store tests. The app
// ships with no seed data, so the fixtures live here in the test file only.
func fixtureState() model.State {
	return model.State{
		GmailAccounts: []model.GmailAccount{
			{
				Email:       "first@example.com",
				Tag:         "Mail Chính",
				Color:       "blue",
				Initials:    "FI",
				Password:    "fixture-password-1",
				Secret2FA:   "KRSXG5CTMVRXEZLU",
				AppPassword: "mnbv cxzl kjhg fdsa",
				BackupCodes: []string{"1122 3344", "5566 7788"},
			},
			{
				Email:     "second@example.com",
				Tag:       "AI & Công Cụ",
				Color:     "purple",
				Initials:  "SE",
				Password:  "fixture-password-2",
				Secret2FA: "JBSWY3DPEHPK3PXP",
			},
		},
		Subscriptions: []model.Subscription{
			{
				ID: 1, Name: "Dịch vụ A", Plan: "Gói A", Category: "AI",
				Gmail: "first@example.com", Price: 20, Cycle: "tháng",
				NextDate: "2026-12-01", Status: model.StatusActive, AuthMode: model.AuthSSO,
			},
			{
				ID: 2, Name: "Dịch vụ B", Plan: "Gói B", Category: "Media",
				Gmail: "first@example.com", Price: 10, Cycle: "tháng",
				NextDate: "2026-12-05", Status: model.StatusActive, AuthMode: model.AuthCustom,
				CustomPass: "fixture-custom-pass",
			},
		},
	}
}

// TestSaveLoadRoundTrip checks that a saved state survives a reload unchanged.
func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	s := New(path)

	want := fixtureState()
	if err := s.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got := s.Load()
	if len(got.GmailAccounts) != len(want.GmailAccounts) {
		t.Fatalf("got %d accounts, want %d", len(got.GmailAccounts), len(want.GmailAccounts))
	}
	if len(got.Subscriptions) != len(want.Subscriptions) {
		t.Fatalf("got %d subscriptions, want %d", len(got.Subscriptions), len(want.Subscriptions))
	}
	if got.GmailAccounts[0].Password != want.GmailAccounts[0].Password {
		t.Errorf("password = %q, want %q", got.GmailAccounts[0].Password, want.GmailAccounts[0].Password)
	}
	if got.Subscriptions[0].CustomPass != want.Subscriptions[0].CustomPass {
		t.Errorf("customPass = %q, want %q", got.Subscriptions[0].CustomPass, want.Subscriptions[0].CustomPass)
	}
}

// TestLoadMissingFileFallsBackToEmpty checks that a first run yields an empty
// vault rather than pre-filled data.
func TestLoadMissingFileFallsBackToEmpty(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "nope", "data.json"))

	got := s.Load()
	if len(got.GmailAccounts) != 0 {
		t.Errorf("got %d accounts, want 0", len(got.GmailAccounts))
	}
	if len(got.Subscriptions) != 0 {
		t.Errorf("got %d subscriptions, want 0", len(got.Subscriptions))
	}
}

// TestLoadCorruptFileFallsBackToEmpty checks that a damaged file does not
// prevent the app from starting.
func TestLoadCorruptFileFallsBackToEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o600); err != nil {
		t.Fatal(err)
	}

	got := New(path).Load()
	if len(got.GmailAccounts) != 0 {
		t.Errorf("got %d accounts, want 0", len(got.GmailAccounts))
	}
}

// TestSaveCreatesDirectory checks that Save creates missing parent directories.
func TestSaveCreatesDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "deeper", "data.json")

	if err := New(path).Save(fixtureState()); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("data file missing: %v", err)
	}
}

// TestSaveLeavesNoTempFiles checks that atomic writes clean up after
// themselves, so the data directory does not accumulate debris.
func TestSaveLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")

	if err := New(path).Save(fixtureState()); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := New(path).Save(fixtureState()); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "data.json" {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("directory contains %v, want only data.json", names)
	}
}

// TestImportExportRoundTrip checks the backup file path used by the UI.
func TestImportExportRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.json")
	want := fixtureState()

	if err := Export(want, path); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	got, err := Import(path)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if len(got.GmailAccounts) != len(want.GmailAccounts) {
		t.Errorf("got %d accounts, want %d", len(got.GmailAccounts), len(want.GmailAccounts))
	}
}

// TestImportRejectsBadFile checks that a file that is not a SubMail backup is
// rejected with an error instead of being loaded as empty state.
func TestImportRejectsBadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte(`{"gmailAccounts":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Import(path); err == nil {
		t.Fatal("Import() expected error for backup missing subscriptionsData")
	}
}

// TestEmptyStateHasNoAccounts checks the shipped state is genuinely empty.
func TestEmptyStateHasNoAccounts(t *testing.T) {
	st := EmptyState()
	if len(st.GmailAccounts) != 0 {
		t.Errorf("got %d accounts, want 0", len(st.GmailAccounts))
	}
	if len(st.Subscriptions) != 0 {
		t.Errorf("got %d subscriptions, want 0", len(st.Subscriptions))
	}
}

// TestFixtureHasNoDuplicateEmails checks the fixture keeps email unique, since
// the email is the account key.
func TestFixtureHasNoDuplicateEmails(t *testing.T) {
	seen := map[string]bool{}
	for _, acc := range fixtureState().GmailAccounts {
		if seen[acc.Email] {
			t.Errorf("duplicate email in fixture: %s", acc.Email)
		}
		seen[acc.Email] = true
	}
}

// TestNormalizeFillsDerivedFields checks that imported data gets the fields
// the frontend relies on, even if the file omitted them.
func TestNormalizeFillsDerivedFields(t *testing.T) {
	st := model.State{
		GmailAccounts: []model.GmailAccount{{Email: "a@b.com"}},
		Subscriptions: []model.Subscription{{ID: 1}},
	}
	st.Normalize()

	if st.GmailAccounts[0].Initials == "" {
		t.Error("Initials not filled in")
	}
	if st.GmailAccounts[0].BackupCodes == nil {
		t.Error("BackupCodes not normalized to an empty slice")
	}
}
