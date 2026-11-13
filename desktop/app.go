package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/phongngo053/submail-personal/internal/integrations"
	"github.com/phongngo053/submail-personal/internal/model"
	"github.com/phongngo053/submail-personal/internal/store"
	"github.com/phongngo053/submail-personal/internal/totp"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Errors returned to the frontend. The frontend maps these to Vietnamese toasts.
var (
	ErrVaultLocked  = errors.New("Két đang khóa!")
	ErrNotFound     = errors.New("Không tìm thấy dữ liệu")
	ErrInvalidInput = errors.New("Dữ liệu không hợp lệ")
)

// VndRate is the fixed USD to VND conversion rate used across the app.
const VndRate = 25400

// App is the Wails-bound application state.
type App struct {
	ctx context.Context

	mu    sync.RWMutex
	store *store.Store
	state model.State
	// locked hides every secret from the frontend. While locked the public
	// payloads carry masks instead of cleartext, and RevealSecret refuses.
	locked bool
}

// NewApp creates the application.
func NewApp() *App {
	return &App{}
}

// startup loads the persisted state when the window opens.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	path, err := store.DefaultPath()
	if err != nil {
		path = "submail-data.json"
	}
	a.store = store.New(path)

	a.mu.Lock()
	defer a.mu.Unlock()
	a.state = a.store.Load()
	// Write the file straight away so the user can always find their data,
	// even if they never edit anything.
	_ = a.store.Save(a.state)
}

// shutdown persists any pending changes.
func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	_ = a.store.Save(a.state)
}

// persist saves the current state to disk. Callers hold the lock.
func (a *App) persist() error {
	return a.store.Save(a.state)
}

// AccountView is a Gmail account as shown in the UI. Secret fields are masked
// while the vault is locked.
type AccountView struct {
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
	IconFile      string   `json:"iconFile"`
}

// SubscriptionView is a subscription as shown in the UI.
type SubscriptionView struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	Plan       string  `json:"plan"`
	Category   string  `json:"category"`
	Gmail      string  `json:"gmail"`
	Price      float64 `json:"price"`
	Cycle      string  `json:"cycle"`
	NextDate   string  `json:"nextDate"`
	NextText   string  `json:"nextText"`
	Urgency    string  `json:"urgency"`
	Payment    string  `json:"payment"`
	ReceiptID  string  `json:"receiptId"`
	Status     string  `json:"status"`
	AuthMode   string  `json:"authMode"`
	CustomPass string  `json:"customPass"`
	IconBg     string  `json:"iconBg"`
	IconText   string  `json:"iconText"`
	IconFile   string  `json:"iconFile"`
}

// Snapshot is the whole app state handed to the frontend in one call.
type Snapshot struct {
	GmailAccounts []AccountView      `json:"gmailAccounts"`
	Subscriptions []SubscriptionView `json:"subscriptionsData"`
	Locked        bool               `json:"locked"`
	VndRate       float64            `json:"vndRate"`
}

// GetState returns the full state for rendering. Secrets are masked while the
// vault is locked.
func (a *App) GetState() Snapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.snapshotLocked()
}

// snapshotLocked builds the public view of the state. Caller holds at least a
// read lock.
func (a *App) snapshotLocked() Snapshot {
	snap := Snapshot{
		GmailAccounts: make([]AccountView, 0, len(a.state.GmailAccounts)),
		Subscriptions: make([]SubscriptionView, 0, len(a.state.Subscriptions)),
		Locked:        a.locked,
		VndRate:       VndRate,
	}

	for _, acc := range a.state.GmailAccounts {
		view := AccountView{
			Email:         acc.Email,
			Tag:           acc.Tag,
			Color:         acc.Color,
			Initials:      acc.Initials,
			RecoveryEmail: acc.RecoveryEmail,
			Phone:         acc.Phone,
			Note:          acc.Note,
			IconFile:      acc.IconFile,
		}
		if !a.locked {
			view.Password = acc.Password
			view.Secret2FA = acc.Secret2FA
			view.AppPassword = acc.AppPassword
			view.BackupCodes = acc.BackupCodes
		} else {
			view.Password = "••••••••••••••"
			view.Secret2FA = "••••••••••••"
			view.AppPassword = "••••••••••••••"
			view.BackupCodes = []string{}
		}
		snap.GmailAccounts = append(snap.GmailAccounts, view)
	}

	now := time.Now()
	for _, sub := range a.state.Subscriptions {
		view := SubscriptionView{
			ID:        sub.ID,
			Name:      sub.Name,
			Plan:      sub.Plan,
			Category:  sub.Category,
			Gmail:     sub.Gmail,
			Price:     sub.Price,
			Cycle:     sub.Cycle,
			NextDate:  sub.NextDate,
			NextText:  renewalText(sub.NextDate, sub.Status, now),
			Urgency:   renewalUrgency(sub.NextDate, sub.Status, now),
			Payment:   sub.Payment,
			ReceiptID: sub.ReceiptID,
			Status:    sub.Status,
			AuthMode:  sub.AuthMode,
			IconBg:    sub.IconBg,
			IconText:  sub.IconText,
			IconFile:  sub.IconFile,
		}
		if !a.locked && sub.AuthMode == model.AuthCustom {
			view.CustomPass = sub.CustomPass
		}
		snap.Subscriptions = append(snap.Subscriptions, view)
	}

	return snap
}

// SetLocked locks or unlocks the vault. Locking hides every secret; unlocking
// is what allows the frontend to ask for them again.
func (a *App) SetLocked(locked bool) Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.locked = locked
	return a.snapshotLocked()
}

// IsLocked reports whether the vault is currently locked.
func (a *App) IsLocked() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.locked
}

// RevealSecret returns one secret for one account. It is refused while the
// vault is locked, so the frontend cannot bypass the lock by reading a payload
// it already holds.
func (a *App) RevealSecret(email, field string) (string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.locked {
		return "", ErrVaultLocked
	}
	for _, acc := range a.state.GmailAccounts {
		if acc.Email != email {
			continue
		}
		switch field {
		case "password":
			return acc.Password, nil
		case "secret2fa":
			return acc.Secret2FA, nil
		case "appPassword":
			return acc.AppPassword, nil
		case "backupCodes":
			return strings.Join(acc.BackupCodes, "\n"), nil
		}
		return "", ErrInvalidInput
	}
	return "", ErrNotFound
}

// RevealCustomPass returns the standalone password of a subscription. It is
// refused while the vault is locked.
func (a *App) RevealCustomPass(id int64) (string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.locked {
		return "", ErrVaultLocked
	}
	for _, sub := range a.state.Subscriptions {
		if sub.ID == id {
			return sub.CustomPass, nil
		}
	}
	return "", ErrNotFound
}

// TotpResult carries a generated code and the seconds left in the time step.
type TotpResult struct {
	Code      string `json:"code"`
	Remaining int    `json:"remaining"`
}

// GetTotp computes the live TOTP code for an account.
func (a *App) GetTotp(email string) (TotpResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, acc := range a.state.GmailAccounts {
		if acc.Email != email {
			continue
		}
		now := time.Now()
		code, err := totp.Code(acc.Secret2FA, now)
		if err != nil {
			return TotpResult{}, ErrInvalidInput
		}
		return TotpResult{Code: code, Remaining: totp.Remaining(now)}, nil
	}
	return TotpResult{}, ErrNotFound
}

// SaveGmail creates or updates a Gmail account.
func (a *App) SaveGmail(acc AccountView) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	email := strings.TrimSpace(acc.Email)
	if email == "" {
		return Snapshot{}, fmt.Errorf("%w: thiếu địa chỉ Gmail", ErrInvalidInput)
	}
	if acc.Color != "" && !model.IsValidColor(acc.Color) {
		return Snapshot{}, fmt.Errorf("%w: màu tài khoản không hợp lệ", ErrInvalidInput)
	}

	for i := range a.state.GmailAccounts {
		existing := &a.state.GmailAccounts[i]
		if existing.Email != email {
			continue
		}
		existing.Tag = acc.Tag
		existing.Color = acc.Color
		existing.Initials = acc.Initials
		existing.RecoveryEmail = acc.RecoveryEmail
		existing.Phone = acc.Phone
		existing.Note = acc.Note
		existing.IconFile = acc.IconFile
		// While the vault is locked the frontend only ever holds masked
		// secrets, so writing them back would destroy the real ones. Keep the
		// stored secrets untouched in that case.
		if !a.locked {
			existing.Password = acc.Password
			existing.Secret2FA = strings.ToUpper(strings.ReplaceAll(acc.Secret2FA, " ", ""))
			existing.AppPassword = acc.AppPassword
			if len(acc.BackupCodes) > 0 {
				existing.BackupCodes = acc.BackupCodes
			}
		}
		if existing.Initials == "" && len(email) >= 2 {
			existing.Initials = email[:2]
		}
		if err := a.persist(); err != nil {
			return Snapshot{}, err
		}
		return a.snapshotLocked(), nil
	}

	// New account: the email must not already exist under a different casing.
	for _, existing := range a.state.GmailAccounts {
		if strings.EqualFold(existing.Email, email) {
			return Snapshot{}, fmt.Errorf("%w: email %s đã tồn tại", ErrInvalidInput, email)
		}
	}

	newAcc := model.GmailAccount{
		Email:         email,
		Tag:           acc.Tag,
		Color:         acc.Color,
		Initials:      acc.Initials,
		Password:      acc.Password,
		Secret2FA:     strings.ToUpper(strings.ReplaceAll(acc.Secret2FA, " ", "")),
		RecoveryEmail: acc.RecoveryEmail,
		Phone:         acc.Phone,
		AppPassword:   acc.AppPassword,
		Note:          acc.Note,
		BackupCodes:   acc.BackupCodes,
		IconFile:      acc.IconFile,
	}
	if newAcc.Initials == "" && len(email) >= 2 {
		newAcc.Initials = email[:2]
	}
	if newAcc.BackupCodes == nil {
		newAcc.BackupCodes = []string{}
	}
	a.state.GmailAccounts = append([]model.GmailAccount{newAcc}, a.state.GmailAccounts...)

	if err := a.persist(); err != nil {
		return Snapshot{}, err
	}
	return a.snapshotLocked(), nil
}

// DeleteGmail removes an account and every subscription that references it.
// The last remaining account can be deleted too, so the user can start over
// from an empty vault.
func (a *App) DeleteGmail(email string) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	kept := a.state.GmailAccounts[:0]
	found := false
	for _, acc := range a.state.GmailAccounts {
		if acc.Email == email {
			found = true
			continue
		}
		kept = append(kept, acc)
	}
	if !found {
		return Snapshot{}, ErrNotFound
	}
	a.state.GmailAccounts = kept

	keptSubs := a.state.Subscriptions[:0]
	for _, sub := range a.state.Subscriptions {
		if sub.Gmail == email {
			continue
		}
		keptSubs = append(keptSubs, sub)
	}
	a.state.Subscriptions = keptSubs

	if err := a.persist(); err != nil {
		return Snapshot{}, err
	}
	return a.snapshotLocked(), nil
}

// SaveSubscription creates or updates a subscription.
func (a *App) SaveSubscription(sub SubscriptionView) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if strings.TrimSpace(sub.Name) == "" {
		return Snapshot{}, fmt.Errorf("%w: thiếu tên dịch vụ", ErrInvalidInput)
	}
	if sub.Category != "" && !model.IsValidCategory(sub.Category) {
		return Snapshot{}, fmt.Errorf("%w: danh mục không hợp lệ", ErrInvalidInput)
	}
	if sub.Status != "" && !model.IsValidStatus(sub.Status) {
		return Snapshot{}, fmt.Errorf("%w: trạng thái không hợp lệ", ErrInvalidInput)
	}
	if sub.AuthMode != "" && !model.IsValidAuthMode(sub.AuthMode) {
		return Snapshot{}, fmt.Errorf("%w: kiểu đăng nhập không hợp lệ", ErrInvalidInput)
	}
	if sub.Gmail == "" {
		return Snapshot{}, fmt.Errorf("%w: chưa chọn Gmail", ErrInvalidInput)
	}

	for i := range a.state.Subscriptions {
		existing := &a.state.Subscriptions[i]
		if existing.ID != sub.ID {
			continue
		}
		existing.Name = sub.Name
		existing.Plan = sub.Plan
		existing.Category = sub.Category
		existing.Gmail = sub.Gmail
		existing.Price = sub.Price
		existing.Cycle = sub.Cycle
		existing.NextDate = sub.NextDate
		existing.Payment = sub.Payment
		existing.ReceiptID = sub.ReceiptID
		existing.Status = sub.Status
		existing.AuthMode = sub.AuthMode
		existing.IconBg = sub.IconBg
		existing.IconText = sub.IconText
		existing.IconFile = sub.IconFile
		// A locked vault never sends the real custom password back, so writing
		// it would erase the stored one.
		if !a.locked {
			existing.CustomPass = sub.CustomPass
		}
		if err := a.persist(); err != nil {
			return Snapshot{}, err
		}
		return a.snapshotLocked(), nil
	}

	newSub := model.Subscription{
		ID:         time.Now().UnixNano(),
		Name:       sub.Name,
		Plan:       sub.Plan,
		Category:   sub.Category,
		Gmail:      sub.Gmail,
		Price:      sub.Price,
		Cycle:      sub.Cycle,
		NextDate:   sub.NextDate,
		Payment:    sub.Payment,
		ReceiptID:  sub.ReceiptID,
		Status:     sub.Status,
		AuthMode:   sub.AuthMode,
		CustomPass: sub.CustomPass,
		IconBg:     sub.IconBg,
		IconText:   sub.IconText,
		IconFile:   sub.IconFile,
	}
	if newSub.Status == "" {
		newSub.Status = model.StatusActive
	}
	if newSub.Cycle == "" {
		newSub.Cycle = "tháng"
	}
	if newSub.ReceiptID == "" {
		newSub.ReceiptID = fmt.Sprintf("SUB-%d", time.Now().UnixNano()%9000+1000)
	}
	a.state.Subscriptions = append([]model.Subscription{newSub}, a.state.Subscriptions...)

	if err := a.persist(); err != nil {
		return Snapshot{}, err
	}
	return a.snapshotLocked(), nil
}

// ToggleSubscriptionPause flips a subscription between active and paused.
// Paused subscriptions are excluded from spend totals and renewal alerts.
func (a *App) ToggleSubscriptionPause(id int64) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := range a.state.Subscriptions {
		sub := &a.state.Subscriptions[i]
		if sub.ID != id {
			continue
		}
		if sub.Status == model.StatusPaused {
			sub.Status = model.StatusActive
		} else {
			sub.Status = model.StatusPaused
		}
		if err := a.persist(); err != nil {
			return Snapshot{}, err
		}
		return a.snapshotLocked(), nil
	}
	return Snapshot{}, ErrNotFound
}

// DeleteSubscription removes a subscription.
func (a *App) DeleteSubscription(id int64) (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	kept := a.state.Subscriptions[:0]
	found := false
	for _, sub := range a.state.Subscriptions {
		if sub.ID == id {
			found = true
			continue
		}
		kept = append(kept, sub)
	}
	if !found {
		return Snapshot{}, ErrNotFound
	}
	a.state.Subscriptions = kept

	if err := a.persist(); err != nil {
		return Snapshot{}, err
	}
	return a.snapshotLocked(), nil
}

// ExportBackup writes the current state to the given path and returns it.
func (a *App) ExportBackup(path string) (string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if err := store.Export(a.state, path); err != nil {
		return "", err
	}
	return path, nil
}

// SaveBackupDialog asks the user where to write a backup and returns that
// path, or an empty string if the dialog was cancelled.
func (a *App) SaveBackupDialog() (string, error) {
	name := fmt.Sprintf("submail-personal-backup-%s.json", time.Now().Format("2006-01-02"))
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Xuất File Sao Lưu",
		DefaultFilename: name,
		Filters: []runtime.FileFilter{
			{DisplayName: "SubMail JSON (*.json)", Pattern: "*.json"},
		},
	})
}

// OpenBackupDialog asks the user for a backup file to import and returns its
// path, or an empty string if the dialog was cancelled.
func (a *App) OpenBackupDialog() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Nhập File Sao Lưu",
		Filters: []runtime.FileFilter{
			{DisplayName: "SubMail JSON (*.json)", Pattern: "*.json"},
		},
	})
}

// ImportBackup replaces the current state with the contents of a backup file.
func (a *App) ImportBackup(path string) (Snapshot, error) {
	loaded, err := store.Import(path)
	if err != nil {
		return Snapshot{}, err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.state = loaded
	if err := a.persist(); err != nil {
		return Snapshot{}, err
	}
	return a.snapshotLocked(), nil
}

// ClearAllData wipes every account and subscription from the vault. The app
// ships without any pre-filled data, so this is the way to start over.
func (a *App) ClearAllData() (Snapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.state = store.EmptyState()
	if err := a.persist(); err != nil {
		return Snapshot{}, err
	}
	return a.snapshotLocked(), nil
}

// DataPath returns the location of the data file, shown in the UI.
func (a *App) DataPath() string {
	return a.store.Path()
}

// IconsDir returns the directory holding the generated icon files.
func (a *App) IconsDir() string {
	if a.store == nil {
		return ""
	}
	return a.store.IconsDir()
}

// GetIntegrations returns the public list of configured integrations. Only the
// id, name, type and enabled flag travel to the frontend: the base URL and the
// API key stay in Go.
func (a *App) GetIntegrations() []integrations.View {
	reg, err := integrations.LoadRegistry()
	if err != nil {
		return []integrations.View{}
	}
	return reg.Views()
}

// CopyToClipboard writes text to the system clipboard.
func (a *App) CopyToClipboard(text string) error {
	if a.ctx == nil {
		return errors.New("ứng dụng chưa sẵn sàng")
	}
	return runtime.ClipboardSetText(a.ctx, text)
}

// renewalText renders the "còn N ngày" label for a renewal date.
func renewalText(nextDate, status string, now time.Time) string {
	if status == model.StatusPaused {
		return "Đã hủy"
	}
	days, ok := daysUntil(nextDate, now)
	if !ok {
		return "Sắp tới"
	}
	switch {
	case days < 0:
		return "Đã quá hạn"
	case days == 0:
		return "Hôm nay"
	case days == 1:
		return "Ngày mai (24h)"
	default:
		return fmt.Sprintf("Còn %d ngày", days)
	}
}

// renewalUrgency classifies a renewal so the UI can colour it.
func renewalUrgency(nextDate, status string, now time.Time) string {
	if status == model.StatusPaused {
		return "normal"
	}
	if status == model.StatusTrial {
		return "urgent"
	}
	days, ok := daysUntil(nextDate, now)
	if !ok {
		return "normal"
	}
	switch {
	case days <= 1:
		return "urgent"
	case days <= 5:
		return "warning"
	default:
		return "normal"
	}
}

// daysUntil returns the whole days from now until the given YYYY-MM-DD date.
func daysUntil(nextDate string, now time.Time) (int, bool) {
	if nextDate == "" {
		return 0, false
	}
	target, err := time.ParseInLocation("2006-01-02", nextDate, now.Location())
	if err != nil {
		return 0, false
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return int(target.Sub(today).Hours() / 24), true
}
