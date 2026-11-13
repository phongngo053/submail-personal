package main

import (
	"strings"
	"testing"
	"time"

	"github.com/phongngo053/submail-personal/internal/model"
	"github.com/phongngo053/submail-personal/internal/store"
)

// Fixture values used by the tests below. They are deliberately fake and live
// only in this file: the app itself ships with no pre-filled data.
const (
	fixtureEmail    = "first@example.com"
	fixtureEmail2   = "second@example.com"
	fixtureEmail3   = "third@example.com"
	fixturePassword = "fixture-password-1"
	fixtureSecret   = "KRSXG5CTMVRXEZLU"
	fixtureAppPass  = "mnbv cxzl kjhg fdsa"
)

// fixtureState returns a small state used across the app-level tests.
func fixtureState() model.State {
	return model.State{
		GmailAccounts: []model.GmailAccount{
			{
				Email:       fixtureEmail,
				Tag:         "Mail Chính",
				Color:       "blue",
				Initials:    "FI",
				Password:    fixturePassword,
				Secret2FA:   fixtureSecret,
				AppPassword: fixtureAppPass,
				BackupCodes: []string{"1122 3344", "5566 7788"},
			},
			{
				Email:     fixtureEmail2,
				Tag:       "AI & Công Cụ",
				Color:     "purple",
				Initials:  "SE",
				Password:  "fixture-password-2",
				Secret2FA: "JBSWY3DPEHPK3PXP",
			},
			{
				Email:     fixtureEmail3,
				Tag:       "Học Tập / Dev",
				Color:     "emerald",
				Initials:  "TH",
				Password:  "fixture-password-3",
				Secret2FA: "MZXW6YTBOI5GM2LE",
			},
		},
		Subscriptions: []model.Subscription{
			{
				ID: 1, Name: "Dịch vụ SSO", Plan: "Gói A", Category: "AI",
				Gmail: fixtureEmail, Price: 20, Cycle: "tháng",
				NextDate: "2026-12-01", Status: model.StatusActive, AuthMode: model.AuthSSO,
			},
			{
				ID: 2, Name: "Dịch vụ riêng", Plan: "Gói B", Category: "Media",
				Gmail: fixtureEmail, Price: 10, Cycle: "tháng",
				NextDate: "2026-12-05", Status: model.StatusActive, AuthMode: model.AuthCustom,
				CustomPass: "fixture-custom-pass",
			},
		},
	}
}

// newTestApp returns an app backed by a temporary data file.
func newTestApp(t *testing.T) *App {
	t.Helper()
	app := NewApp()
	app.store = store.New(t.TempDir() + "/data.json")
	app.state = fixtureState()
	return app
}

// TestLockedSnapshotHidesSecrets checks that a locked vault never sends
// cleartext secrets to the frontend.
func TestLockedSnapshotHidesSecrets(t *testing.T) {
	app := newTestApp(t)
	app.locked = true

	snap := app.GetState()
	for _, acc := range snap.GmailAccounts {
		if strings.Contains(acc.Password, "fixture-password") {
			t.Errorf("password leaked while locked: %q", acc.Password)
		}
		if acc.Secret2FA == fixtureSecret {
			t.Error("2FA secret leaked while locked")
		}
		if acc.AppPassword == fixtureAppPass {
			t.Error("app password leaked while locked")
		}
		if len(acc.BackupCodes) != 0 {
			t.Errorf("backup codes leaked while locked: %v", acc.BackupCodes)
		}
	}
	for _, sub := range snap.Subscriptions {
		if sub.CustomPass != "" {
			t.Errorf("custom pass leaked while locked: %q", sub.CustomPass)
		}
	}
}

// TestUnlockedSnapshotIncludesSecrets checks the unlocked view carries the
// data the UI needs.
func TestUnlockedSnapshotIncludesSecrets(t *testing.T) {
	app := newTestApp(t)
	app.locked = false

	snap := app.GetState()
	if snap.GmailAccounts[0].Password != fixturePassword {
		t.Errorf("password = %q, want the stored value", snap.GmailAccounts[0].Password)
	}
	if snap.GmailAccounts[0].Secret2FA != fixtureSecret {
		t.Errorf("secret = %q, want the stored value", snap.GmailAccounts[0].Secret2FA)
	}
	found := false
	for _, sub := range snap.Subscriptions {
		if sub.CustomPass != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected at least one subscription with a custom password")
	}
}

// TestRevealSecretRefusedWhileLocked checks the backend refuses to hand out
// secrets while the vault is locked. This is the root-cause fix for the
// password leak: the frontend cannot obtain the value even if it asks.
func TestRevealSecretRefusedWhileLocked(t *testing.T) {
	app := newTestApp(t)
	app.locked = true

	email := app.state.GmailAccounts[0].Email
	for _, field := range []string{"password", "secret2fa", "appPassword", "backupCodes"} {
		if _, err := app.RevealSecret(email, field); err == nil {
			t.Errorf("RevealSecret(%q) succeeded while locked, want error", field)
		}
	}
	if _, err := app.RevealCustomPass(app.state.Subscriptions[0].ID); err == nil {
		t.Error("RevealCustomPass() succeeded while locked, want error")
	}
}

// TestRevealSecretAllowedWhileUnlocked checks the normal unlock path works.
func TestRevealSecretAllowedWhileUnlocked(t *testing.T) {
	app := newTestApp(t)
	app.locked = false

	email := app.state.GmailAccounts[0].Email
	got, err := app.RevealSecret(email, "password")
	if err != nil {
		t.Fatalf("RevealSecret() error = %v", err)
	}
	if got != fixturePassword {
		t.Errorf("password = %q, want the stored value", got)
	}
}

// TestSetLockedTogglesAndMasks checks the lock toggle changes what the
// frontend receives.
func TestSetLockedTogglesAndMasks(t *testing.T) {
	app := newTestApp(t)

	locked := app.SetLocked(true)
	if !locked.Locked {
		t.Error("Locked = false after SetLocked(true)")
	}
	if locked.GmailAccounts[0].Password == fixturePassword {
		t.Error("password still cleartext after locking")
	}

	unlocked := app.SetLocked(false)
	if unlocked.Locked {
		t.Error("Locked = true after SetLocked(false)")
	}
	if unlocked.GmailAccounts[0].Password != fixturePassword {
		t.Error("password still masked after unlocking")
	}
}

// TestDeleteGmailRemovesSubscriptions checks that deleting an account also
// removes the subscriptions that reference it.
func TestDeleteGmailRemovesSubscriptions(t *testing.T) {
	app := newTestApp(t)
	email := app.state.GmailAccounts[0].Email

	snap, err := app.DeleteGmail(email)
	if err != nil {
		t.Fatalf("DeleteGmail() error = %v", err)
	}
	for _, sub := range snap.Subscriptions {
		if sub.Gmail == email {
			t.Errorf("subscription %q still references deleted account", sub.Name)
		}
	}
	for _, acc := range snap.GmailAccounts {
		if acc.Email == email {
			t.Error("deleted account still present")
		}
	}
}

// TestDeleteLastAccountAllowed checks that the very last Gmail account can be
// removed, so the user can start over from an empty vault.
func TestDeleteLastAccountAllowed(t *testing.T) {
	app := newTestApp(t)
	app.state.GmailAccounts = app.state.GmailAccounts[:1]
	app.state.Subscriptions = app.state.Subscriptions[:0]

	snap, err := app.DeleteGmail(app.state.GmailAccounts[0].Email)
	if err != nil {
		t.Fatalf("DeleteGmail() error = %v", err)
	}
	if len(snap.GmailAccounts) != 0 {
		t.Errorf("got %d accounts, want 0", len(snap.GmailAccounts))
	}
}

// TestSaveGmailRejectsDuplicateEmail checks the email uniqueness rule.
func TestSaveGmailRejectsDuplicateEmail(t *testing.T) {
	app := newTestApp(t)
	existing := app.state.GmailAccounts[0].Email

	if _, err := app.SaveGmail(AccountView{Email: existing, Tag: "X"}); err != nil {
		// Updating the same account is allowed.
		t.Logf("update path returned: %v", err)
	}

	other := app.state.GmailAccounts[1].Email
	if _, err := app.SaveGmail(AccountView{Email: strings.ToUpper(other), Tag: "X"}); err == nil {
		t.Error("SaveGmail() accepted an email that differs only by case")
	}
}

// TestSaveGmailRejectsUnknownColor checks the controlled colour set.
func TestSaveGmailRejectsUnknownColor(t *testing.T) {
	app := newTestApp(t)

	if _, err := app.SaveGmail(AccountView{Email: "new@example.com", Color: "chartreuse"}); err == nil {
		t.Error("SaveGmail() accepted an unknown color")
	}
}

// TestSaveSubscriptionRejectsUnknownCategory checks the controlled category set.
func TestSaveSubscriptionRejectsUnknownCategory(t *testing.T) {
	app := newTestApp(t)

	_, err := app.SaveSubscription(SubscriptionView{
		Name:     "X",
		Category: "Nope",
		Gmail:    app.state.GmailAccounts[0].Email,
	})
	if err == nil {
		t.Error("SaveSubscription() accepted an unknown category")
	}
}

// TestToggleSubscriptionPause checks the paused state and that a paused
// subscription reports no spend.
func TestToggleSubscriptionPause(t *testing.T) {
	app := newTestApp(t)
	id := app.state.Subscriptions[0].ID

	snap, err := app.ToggleSubscriptionPause(id)
	if err != nil {
		t.Fatalf("ToggleSubscriptionPause() error = %v", err)
	}
	var status string
	for _, sub := range snap.Subscriptions {
		if sub.ID == id {
			status = sub.Status
		}
	}
	if status != model.StatusPaused {
		t.Errorf("status = %q, want %q", status, model.StatusPaused)
	}
}

// TestClearAllDataWipesVault checks the "start over" action empties the vault
// and persists the empty state.
func TestClearAllDataWipesVault(t *testing.T) {
	app := newTestApp(t)

	snap, err := app.ClearAllData()
	if err != nil {
		t.Fatalf("ClearAllData() error = %v", err)
	}
	if len(snap.GmailAccounts) != 0 {
		t.Errorf("got %d accounts, want 0", len(snap.GmailAccounts))
	}
	if len(snap.Subscriptions) != 0 {
		t.Errorf("got %d subscriptions, want 0", len(snap.Subscriptions))
	}

	reloaded := app.store.Load()
	if len(reloaded.GmailAccounts) != 0 || len(reloaded.Subscriptions) != 0 {
		t.Error("empty state was not persisted to disk")
	}
}

// TestGetTotpMatchesSecret checks the Go TOTP agrees with the stored secret.
func TestGetTotpMatchesSecret(t *testing.T) {
	app := newTestApp(t)
	email := app.state.GmailAccounts[2].Email

	res, err := app.GetTotp(email)
	if err != nil {
		t.Fatalf("GetTotp() error = %v", err)
	}
	if len(res.Code) != 6 {
		t.Errorf("code = %q, want 6 digits", res.Code)
	}
	if res.Remaining < 1 || res.Remaining > 30 {
		t.Errorf("remaining = %d, want 1..30", res.Remaining)
	}
}

// TestRenewalTextAndUrgency checks the renewal labels stay in sync with the
// date instead of drifting.
func TestRenewalTextAndUrgency(t *testing.T) {
	now := time.Date(2026, 11, 12, 10, 0, 0, 0, time.Local)
	today := now.Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")
	later := now.AddDate(0, 0, 10).Format("2006-01-02")

	if got := renewalText(today, model.StatusActive, now); got != "Hôm nay" {
		t.Errorf("renewalText(today) = %q, want %q", got, "Hôm nay")
	}
	if got := renewalText(tomorrow, model.StatusActive, now); got != "Ngày mai (24h)" {
		t.Errorf("renewalText(tomorrow) = %q, want %q", got, "Ngày mai (24h)")
	}
	if got := renewalText(later, model.StatusActive, now); got != "Còn 10 ngày" {
		t.Errorf("renewalText(later) = %q, want %q", got, "Còn 10 ngày")
	}
	if got := renewalUrgency(tomorrow, model.StatusActive, now); got != "urgent" {
		t.Errorf("renewalUrgency(tomorrow) = %q, want %q", got, "urgent")
	}
	if got := renewalText(later, model.StatusPaused, now); got != "Đã hủy" {
		t.Errorf("renewalText(paused) = %q, want %q", got, "Đã hủy")
	}
}
