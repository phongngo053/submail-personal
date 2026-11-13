// Package model defines the persisted data structures for SubMail Personal.
package model

// GmailAccount is one personal Gmail inbox with its credentials and 2FA secret.
type GmailAccount struct {
	Email         string   `json:"email"`
	Tag           string   `json:"tag"`
	Color         string   `json:"color"`
	Initials      string   `json:"initials"`
	Password      string   `json:"password"`
	Secret2FA     string   `json:"secret2fa"`
	RecoveryEmail string   `json:"recoveryEmail"`
	Phone         string   `json:"phone"`
	AppPassword   string   `json:"appPassword"`
	Note          string   `json:"note"`
	BackupCodes   []string `json:"backupCodes"`
	// IconFile is the file name of the AI-generated avatar inside the icons
	// directory. Empty means the account still uses its initials.
	IconFile string `json:"iconFile"`
}

// Subscription is a paid service registered under a Gmail account.
type Subscription struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	Plan       string  `json:"plan"`
	Category   string  `json:"category"`
	Gmail      string  `json:"gmail"`
	Price      float64 `json:"price"`
	Cycle      string  `json:"cycle"`
	NextDate   string  `json:"nextDate"`
	Payment    string  `json:"payment"`
	ReceiptID  string  `json:"receiptId"`
	Status     string  `json:"status"`
	AuthMode   string  `json:"authMode"`
	CustomPass string  `json:"customPass"`
	IconBg     string  `json:"iconBg"`
	IconText   string  `json:"iconText"`
	// IconFile is the file name of the AI-generated icon inside the icons
	// directory. Empty means the subscription still uses its text badge.
	IconFile string `json:"iconFile"`
}

// Subscription status values.
const (
	StatusActive = "active"
	StatusTrial  = "trial"
	StatusPaused = "paused"
)

// Auth modes: sso reuses the Gmail login, custom has its own password.
const (
	AuthSSO    = "sso"
	AuthCustom = "custom"
)

// Categories available for a subscription.
var Categories = []string{"AI", "Media", "Dev", "Storage", "Shopping"}

// IsValidCategory reports whether c is one of the known categories.
func IsValidCategory(c string) bool {
	for _, known := range Categories {
		if c == known {
			return true
		}
	}
	return false
}

// IsValidStatus reports whether s is a known subscription status.
func IsValidStatus(s string) bool {
	return s == StatusActive || s == StatusTrial || s == StatusPaused
}

// IsValidAuthMode reports whether m is a known auth mode.
func IsValidAuthMode(m string) bool {
	return m == AuthSSO || m == AuthCustom
}

// AccountColors are the valid avatar colors. A missing entry in a colour map
// renders "undefined" in the UI, so the set is closed.
var AccountColors = []string{"blue", "purple", "amber", "emerald", "rose"}

// IsValidColor reports whether c is a known account color.
func IsValidColor(c string) bool {
	for _, known := range AccountColors {
		if c == known {
			return true
		}
	}
	return false
}

// State is the whole persisted document.
type State struct {
	GmailAccounts []GmailAccount `json:"gmailAccounts"`
	Subscriptions []Subscription `json:"subscriptionsData"`
}

// NewState returns an empty state with non-nil slices so JSON round-trips cleanly.
func NewState() State {
	return State{
		GmailAccounts: []GmailAccount{},
		Subscriptions: []Subscription{},
	}
}

// Normalize replaces nil slices with empty ones and fills in derived fields so a
// hand-edited or imported file cannot produce nil dereferences in the frontend.
func (s *State) Normalize() {
	if s.GmailAccounts == nil {
		s.GmailAccounts = []GmailAccount{}
	}
	if s.Subscriptions == nil {
		s.Subscriptions = []Subscription{}
	}
	for i := range s.GmailAccounts {
		acc := &s.GmailAccounts[i]
		if acc.BackupCodes == nil {
			acc.BackupCodes = []string{}
		}
		if acc.Initials == "" && len(acc.Email) >= 2 {
			acc.Initials = acc.Email[:2]
		}
	}
}
