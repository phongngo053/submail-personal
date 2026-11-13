import { esc, escJs } from './escape.js';
import {
  serviceIconHtml,
  avatarHtml,
  avatarClass,
  genIconButtonHtml as buildGenIconButtonHtml,
  presetCardHtml,
} from './icons.js';

// The Go backend owns the data. This module only renders it and sends user
// actions back. Secrets are never cached here: while the vault is locked the
// backend returns masks, and RevealSecret is refused server-side.

let gmailAccounts = [];
let subscriptionsData = [];
let vaultLocked = false;
let vndRate = 25400;
// True when the user configured an enabled image service on this machine.
// The backend decides; the button only appears when it is available.
let imageServiceReady = false;

let selectedGmail = '';
let vaultTagFilter = 'all';
let overviewGmailFilter = 'all';
let showDetailPassword = false;
let quickVaultTarget = null;
let activePage = 'vault';
let globalQuery = '';

// Account accent colours, keyed by the persisted `color` field.
const pillClassMap = {
  blue: 'pill-blue',
  purple: 'pill-purple',
  amber: 'pill-amber',
  emerald: 'pill-emerald',
  rose: 'pill-rose',
};
const barClassMap = {
  blue: 'bar-blue',
  purple: 'bar-purple',
  amber: 'bar-amber',
  emerald: 'bar-emerald',
  rose: 'bar-rose',
};

const PAGE_META = {
  vault: { title: 'Két Gmail', subtitle: 'Mật khẩu, mã 2FA sống và gói đăng ký' },
  overview: { title: 'Tổng quan Subscriptions', subtitle: 'Chi phí, gia hạn và cảnh báo toàn bộ' },
  system: { title: 'Hệ thống', subtitle: 'Dữ liệu, sao lưu và bảo mật' },
};

function formatVnd(usd) {
  return Math.round(usd * vndRate).toLocaleString('vi-VN') + 'đ';
}

// ------------------------------------------------------------------ backend

/**
 * Calls a bound Go method and surfaces failures as a Vietnamese toast.
 * @param {string} method
 * @param {...unknown} args
 */
async function call(method, ...args) {
  try {
    return await window.go.main.App[method](...args);
  } catch (err) {
    showToast(String(err?.message || err || 'Có lỗi xảy ra'), 'error');
    return null;
  }
}

/** Applies a snapshot returned by the backend to the local view state. */
function applySnapshot(snap) {
  if (!snap) return;
  gmailAccounts = snap.gmailAccounts || [];
  subscriptionsData = snap.subscriptionsData || [];
  vaultLocked = !!snap.locked;
  vndRate = snap.vndRate || 25400;
  if (!gmailAccounts.some((a) => a.email === selectedGmail)) {
    selectedGmail = gmailAccounts[0]?.email || '';
  }
}

/** Refreshes the whole UI from the backend. */
async function reload() {
  applySnapshot(await call('GetState'));
  refreshAll();
}

// ------------------------------------------------------------------- icons

// Icons are stored as PNG files next to the app data. The frontend asks the
// backend for them once and keeps the result, so a table re-render does not
// fetch the same picture again.
const iconCache = new Map();

/**
 * Fetches one icon as a data URI, using the cache when possible.
 * @param {string} file
 * @returns {Promise<string|null>}
 */
async function loadIcon(file) {
  if (!file) return null;
  if (iconCache.has(file)) return iconCache.get(file);
  const dataUri = await call('GetIconData', file);
  if (!dataUri) return null;
  iconCache.set(file, dataUri);
  return dataUri;
}

/**
 * Replaces the text badge of every icon holder with its AI generated picture.
 * Elements without a picture keep their badge.
 * @param {ParentNode} [root]
 */
async function hydrateIcons(root = document) {
  const holders = root.querySelectorAll('[data-icon-file]');
  const pending = [];
  for (const holder of holders) {
    const file = holder.dataset.iconFile;
    if (!file || holder.dataset.iconLoaded === file) continue;
    pending.push(
      loadIcon(file).then((dataUri) => {
        if (!dataUri) return;
        const img = document.createElement('img');
        img.src = dataUri;
        img.alt = '';
        holder.dataset.iconLoaded = file;
        holder.replaceChildren(img);
      })
    );
  }
  await Promise.all(pending);
}

/**
 * Builds the "Tạo icon AI" button for a subscription. The button only appears
 * when the user configured an image service on this machine, so a fresh
 * install shows no dead action.
 * @param {object} sub
 * @returns {string}
 */
function genIconButtonHtml(sub) {
  return buildGenIconButtonHtml(sub, imageServiceReady);
}

// ------------------------------------------------------------------ helpers

function findAccount(email) {
  return gmailAccounts.find((a) => a.email === email);
}

/** Active subscriptions are the only ones counted in spend and alerts. */
function activeSubscriptions() {
  return subscriptionsData.filter((s) => s.status !== 'paused');
}

/** Total monthly spend across every active subscription. */
function totalSpend() {
  return activeSubscriptions().reduce((sum, s) => sum + s.price, 0);
}

/** Subscriptions that need attention soon (trial or renewal within 5 days). */
function expiringSubscriptions() {
  return activeSubscriptions().filter((s) => s.urgency === 'urgent' || s.urgency === 'warning');
}

/**
 * Copies text to the system clipboard through the backend.
 * @param {string} text
 * @param {string} msg
 */
async function copyString(text, msg) {
  if (!text) {
    showToast('Không có dữ liệu để sao chép!', 'error');
    return;
  }
  const err = await call('CopyToClipboard', text);
  if (err === null) return;
  showToast(msg || 'Đã sao chép!', 'ok');
}

/**
 * Asks the backend for a secret. The backend refuses while the vault is
 * locked, so the lock cannot be bypassed from the UI.
 * @param {string} email
 * @param {string} field
 * @param {string} msg
 */
async function copyAccountSecret(email, field, msg) {
  if (vaultLocked) {
    showToast('Két đang khóa!', 'error');
    return;
  }
  const secret = await call('RevealSecret', email, field);
  if (secret === null || secret === undefined) return;
  await copyString(secret, msg);
}

/**
 * Asks the backend for a subscription's own password.
 * @param {number} id
 * @param {string} name
 */
async function copyCustomPass(id, name) {
  if (vaultLocked) {
    showToast('Két đang khóa!', 'error');
    return;
  }
  const secret = await call('RevealCustomPass', id);
  if (secret === null || secret === undefined) return;
  await copyString(secret, `Đã sao chép mật khẩu riêng của ${name}`);
}

/** Copies the text content of an element by its id. */
function copyTextById(id, msg) {
  const el = document.getElementById(id);
  if (!el) return;
  copyString(el.innerText, msg);
}

// ------------------------------------------------------------------ TOTP

async function tickTotp() {
  const rem = 30 - (Math.floor(Date.now() / 1000) % 30);
  const pct = (rem / 30) * 100;
  const globalTimer = document.getElementById('globalTotpTimer');
  if (globalTimer) globalTimer.innerText = `${rem}s`;

  const acc = findAccount(selectedGmail);
  if (acc) {
    const codeEl = document.getElementById('detailTotpCode');
    const timerEl = document.getElementById('detailTotpCountdown');
    const barEl = document.getElementById('detailTotpProgressBar');

    if (timerEl) timerEl.innerText = `${rem}s`;
    if (barEl) {
      barEl.style.width = `${pct}%`;
      barEl.classList.toggle('is-urgent', rem <= 5);
    }
    if (timerEl) timerEl.classList.toggle('is-urgent', rem <= 5);

    if (codeEl) {
      if (vaultLocked) {
        codeEl.innerText = '••• •••';
      } else {
        const res = await call('GetTotp', selectedGmail);
        codeEl.innerText = res ? `${res.code.slice(0, 3)} ${res.code.slice(3)}` : '••• •••';
      }
    }
  }

  if (quickVaultTarget) {
    const qAcc = findAccount(quickVaultTarget.gmail);
    if (qAcc) {
      const timer = document.getElementById('qvTimer');
      const code = document.getElementById('qvTotp');
      if (timer) timer.innerText = `${rem}s`;
      if (code) {
        if (vaultLocked) {
          code.innerText = '••• •••';
        } else {
          const res = await call('GetTotp', quickVaultTarget.gmail);
          code.innerText = res ? `${res.code.slice(0, 3)} ${res.code.slice(3)}` : '••• •••';
        }
      }
    }
  }
}

setInterval(tickTotp, 1000);

// ------------------------------------------------------------------ chrome

/** Switches the light/dark theme and remembers the choice. */
function toggleTheme() {
  const html = document.documentElement;
  const isDark = html.classList.toggle('dark');
  try {
    localStorage.setItem('darkMode', JSON.stringify(isDark));
  } catch {
    /* storage unavailable */
  }
}

/** Reads the page name from the location hash, falling back to the vault. */
function currentPage() {
  const hash = window.location.hash.replace(/^#\/?/, '');
  return PAGE_META[hash] ? hash : 'vault';
}

/** Navigates to one of the three pages and syncs the sidebar + topbar. */
function switchMainView(view) {
  if (!PAGE_META[view]) return;
  activePage = view;

  document.querySelectorAll('.page').forEach((section) => {
    section.classList.toggle('is-active', section.dataset.page === view);
  });

  document.querySelectorAll('.menu-item[data-nav]').forEach((item) => {
    item.classList.toggle('is-active', item.dataset.nav === view);
  });

  const meta = PAGE_META[view];
  document.getElementById('pageTitle').innerText = meta.title;
  document.getElementById('pageSubtitle').innerText = meta.subtitle;

  refreshAll();
  closeSidebar();
  document.querySelector('.content-wrap')?.scrollTo({ top: 0 });
}

/** Closes the mobile drawer and clears the toggle state. */
function closeSidebar() {
  const sidebar = document.getElementById('sidebar');
  if (!sidebar) return;
  sidebar.classList.remove('is-open');
  document.getElementById('sidebarOverlay').classList.remove('is-open');
  document.getElementById('sidebarToggle').setAttribute('aria-expanded', 'false');
}

/** Opens or closes the sidebar. Wide layouts collapse it instead. */
function toggleSidebar() {
  if (window.innerWidth >= 1280) {
    document.body.classList.toggle('sidebar-collapsed');
    return;
  }
  const sidebar = document.getElementById('sidebar');
  const overlay = document.getElementById('sidebarOverlay');
  const open = !sidebar.classList.contains('is-open');
  sidebar.classList.toggle('is-open', open);
  overlay.classList.toggle('is-open', open);
  document.getElementById('sidebarToggle').setAttribute('aria-expanded', String(open));
}

async function toggleVaultLock() {
  const nextLocked = !vaultLocked;
  applySnapshot(await call('SetLocked', nextLocked));
  if (vaultLocked) showDetailPassword = false;
  refreshAll();
  showToast(
    vaultLocked
      ? 'Đã khóa Két — Mật khẩu & mã 2FA đã được che.'
      : 'Đã mở khóa Két.',
    vaultLocked ? 'error' : 'ok'
  );
}

// ------------------------------------------------------------------ vault list

function setVaultTagFilter(tag) {
  vaultTagFilter = tag;
  renderVaultTagPills();
  renderVaultAccountList();
}

/** Renders the tag filter pills derived from the accounts in the state. */
function renderVaultTagPills() {
  const container = document.getElementById('vaultTagPills');
  const tags = [];
  gmailAccounts.forEach((acc) => {
    if (acc.tag && !tags.includes(acc.tag)) tags.push(acc.tag);
  });

  const entries = [['all', 'Tất cả'], ...tags.map((t) => [t, t])];
  container.innerHTML = entries
    .map(
      ([value, label]) =>
        `<button type="button" class="filter-pill ${vaultTagFilter === value ? 'is-active' : ''}" data-vtag="${esc(value)}" onclick="setVaultTagFilter('${escJs(value)}')">${esc(label)}</button>`
    )
    .join('');
}

function renderVaultAccountList() {
  const searchEl = document.getElementById('vaultSearchGmail');
  const search = (searchEl?.value || '').toLowerCase().trim();
  const container = document.getElementById('vaultAccountList');

  const filtered = gmailAccounts.filter((acc) => {
    if (vaultTagFilter !== 'all' && acc.tag !== vaultTagFilter) return false;
    if (!search) return true;
    return (
      acc.email.toLowerCase().includes(search) ||
      acc.tag.toLowerCase().includes(search) ||
      (acc.note || '').toLowerCase().includes(search)
    );
  });

  if (filtered.length === 0) {
    container.innerHTML =
      gmailAccounts.length === 0
        ? '<div class="empty">Chưa có tài khoản Gmail nào.<br /><strong>Nhấn “Thêm Gmail” để bắt đầu.</strong></div>'
        : '<div class="empty">Không tìm thấy Gmail nào.<br /><strong>Thử từ khoá khác.</strong></div>';
    return;
  }

  container.innerHTML = filtered
    .map((acc) => {
      const subs = activeSubscriptions().filter((s) => s.gmail === acc.email);
      const spend = subs.reduce((sum, s) => sum + s.price, 0);
      const isSelected = acc.email === selectedGmail;
      const avatarCls = avatarClass(acc.color);
      const pillCls = pillClassMap[acc.color] || 'pill-blue';

      return `
          <button type="button" onclick="selectVaultAccount('${escJs(acc.email)}')" class="vault-item ${isSelected ? 'is-active' : ''}">
            <div class="vault-item-main">
              <div class="avatar ${avatarCls}" data-icon-file="${esc(acc.iconFile || '')}">${esc(
                acc.initials
              )}</div>
              <div style="min-width: 0">
                <div class="vault-item-email mono">${esc(acc.email)}</div>
                <div class="vault-item-meta">
                  <span class="tag-pill ${pillCls}">${esc(acc.tag)}</span>
                  <span class="twofa-on">2FA ON</span>
                </div>
              </div>
            </div>
            <div class="vault-item-spend">
              <b class="tabular-nums">$${spend.toFixed(2)}</b>
              <span class="tabular-nums">${subs.length} gói</span>
            </div>
          </button>
        `;
    })
    .join('');
}

function selectVaultAccount(email) {
  selectedGmail = email;
  showDetailPassword = false;
  renderVaultAccountList();
  renderVaultDetail();
  tickTotp();
}

// ------------------------------------------------------------------ vault detail

function renderVaultDetail() {
  const emptyState = document.getElementById('vaultEmptyState');
  const detailContent = document.getElementById('vaultDetailContent');
  const acc = findAccount(selectedGmail) || gmailAccounts[0];

  // The vault ships empty: show a call to action instead of a blank detail pane.
  if (!acc) {
    if (emptyState) emptyState.classList.remove('hidden');
    if (detailContent) detailContent.classList.add('hidden');
    return;
  }
  if (emptyState) emptyState.classList.add('hidden');
  if (detailContent) detailContent.classList.remove('hidden');

  selectedGmail = acc.email;

  const avatarCls = avatarClass(acc.color);
  const pillCls = pillClassMap[acc.color] || 'pill-blue';

  const avatarEl = document.getElementById('detailAvatar');
  avatarEl.className = `avatar avatar-lg ${avatarCls}`;
  avatarEl.innerText = acc.initials;
  avatarEl.dataset.iconFile = acc.iconFile || '';
  delete avatarEl.dataset.iconLoaded;

  document.getElementById('detailEmail').innerText = acc.email;
  const tagBadge = document.getElementById('detailTagBadge');
  tagBadge.className = `tag-pill ${pillCls}`;
  tagBadge.innerText = acc.tag;

  document.getElementById('detailRecoveryDisplay').innerText = `Khôi phục: ${
    acc.recoveryEmail || 'Chưa cài đặt'
  }`;
  document.getElementById('detailNoteDisplay').innerText = acc.note || 'Không có ghi chú';
  document.getElementById('detailPhoneDisplay').innerText = acc.phone || 'Chưa liên kết';

  document.getElementById('detailPasswordText').innerText =
    vaultLocked || !showDetailPassword ? '••••••••••••••' : acc.password;

  const sec = acc.secret2fa || 'NONE';
  document.getElementById('detailSecretMask').innerText = vaultLocked
    ? '••••••••••••'
    : `${sec.slice(0, 4)}••••${sec.slice(-4)}`;

  const subs = subscriptionsData.filter((s) => s.gmail === acc.email);
  const activeSubs = subs.filter((s) => s.status !== 'paused');
  const total = activeSubs.reduce((sum, s) => sum + s.price, 0);

  document.getElementById('detailSubCountBadge').innerText = `${activeSubs.length} gói`;
  document.getElementById('detailSubTotalSpend').innerText = `$${total.toFixed(2)}/tháng`;
  document.getElementById('detailSubTotalVnd').innerText = `(~${formatVnd(total)})`;

  const tbody = document.getElementById('detailSubsTableBody');
  if (subs.length === 0) {
    tbody.innerHTML =
      '<tr><td colspan="5"><div class="empty">Gmail này chưa có gói nào.<br /><strong>Nhấn “Thêm Sub” để thêm.</strong></div></td></tr>';
    return;
  }

  tbody.innerHTML = subs
    .map((sub) => {
      const authHtml =
        sub.authMode === 'sso'
          ? '<span class="sso-badge">Google SSO</span>'
          : `<div class="row" style="gap: 6px">
               <span class="mono pass-chip">${vaultLocked ? '••••••' : esc(sub.customPass)}</span>
               <button type="button" class="link-btn" onclick="copyCustomPass(${sub.id}, '${escJs(
                 sub.name
               )}')">Copy</button>
             </div>`;

      let renewalHtml = `<span class="renewal-badge renewal-normal">${esc(sub.nextText)}</span>`;
      if (sub.status === 'paused') {
        renewalHtml = '<span class="renewal-badge renewal-normal">Đã hủy</span>';
      } else if (sub.urgency === 'urgent') {
        renewalHtml = `<span class="renewal-badge renewal-urgent">${esc(sub.nextText)}</span>`;
      } else if (sub.urgency === 'warning') {
        renewalHtml = `<span class="renewal-badge renewal-warning">${esc(sub.nextText)}</span>`;
      }

      const pauseBtn =
        sub.status !== 'paused'
          ? `<button type="button" class="link-btn danger" onclick="toggleSubPause(${sub.id})">Hủy</button>`
          : `<button type="button" class="link-btn" style="color: var(--success-600)" onclick="toggleSubPause(${sub.id})">Mở lại</button>`;

      return `
          <tr class="${sub.status === 'paused' ? 'is-paused' : ''}">
            <td>
              <div class="cell-main">
                ${serviceIconHtml(sub)}
                <div style="min-width: 0">
                  <div class="cell-title">${esc(sub.name)}</div>
                  <div class="cell-sub">${esc(sub.plan)}</div>
                </div>
              </div>
            </td>
            <td>${authHtml}</td>
            <td class="text-right">
              <div class="cell-title tabular-nums">$${sub.price.toFixed(2)}<span
                  class="cell-sub"
                  style="font-weight: 400"
                  >/${esc(sub.cycle)}</span
                ></div>
              <div class="cell-sub tabular-nums">~${formatVnd(sub.price)}</div>
            </td>
            <td>
              <div>${renewalHtml}</div>
              <div class="cell-sub">${esc(sub.nextDate)}</div>
            </td>
            <td class="text-right">
              <div class="actions" style="justify-content: flex-end">
                <button type="button" class="link-btn" onclick="previewReceiptModal('${escJs(
                  sub.name
                )}', '$${sub.price.toFixed(2)} (~${formatVnd(sub.price)})', '${escJs(
                  sub.receiptId
                )}', '${escJs(sub.gmail)}', '${escJs(sub.payment)}', '${escJs(
                  sub.nextDate
                )}')">Biên lai</button>
                ${genIconButtonHtml(sub)}
                ${pauseBtn}
                <button type="button" class="link-btn danger" onclick="deleteSub(${sub.id})" title="Xóa hẳn">✕</button>
              </div>
            </td>
          </tr>
        `;
    })
    .join('');
}

function toggleDetailPassword() {
  if (vaultLocked) {
    showToast('Hãy mở khóa Két trước khi xem mật khẩu!', 'error');
    return;
  }
  showDetailPassword = !showDetailPassword;
  renderVaultDetail();
}

async function copySelectedPassword() {
  await copyAccountSecret(selectedGmail, 'password', `Đã sao chép mật khẩu của ${selectedGmail}`);
}

async function copySelectedTotp() {
  if (vaultLocked) {
    showToast('Két đang khóa!', 'error');
    return;
  }
  const res = await call('GetTotp', selectedGmail);
  if (!res) return;
  await copyString(res.code, `Đã sao chép mã 2FA: ${res.code}`);
}

async function copySecretKey() {
  await copyAccountSecret(selectedGmail, 'secret2fa', 'Đã sao chép Secret Key 2FA');
}

async function copyAppPassword() {
  await copyAccountSecret(selectedGmail, 'appPassword', 'Đã sao chép App Password');
}

async function copyAllBackupCodes() {
  await copyAccountSecret(selectedGmail, 'backupCodes', 'Đã sao chép toàn bộ 8 mã dự phòng!');
}

async function deleteSelectedGmail() {
  if (gmailAccounts.length <= 1) {
    showToast('Cần giữ lại ít nhất 1 tài khoản Gmail!', 'error');
    return;
  }
  if (!confirm(`Xóa tài khoản ${selectedGmail} và các gói đăng ký của nó?`)) return;
  applySnapshot(await call('DeleteGmail', selectedGmail));
  selectedGmail = gmailAccounts[0]?.email || '';
  refreshAll();
  showToast('Đã xóa tài khoản Gmail!', 'ok');
}

async function toggleSubPause(id) {
  applySnapshot(await call('ToggleSubscriptionPause', id));
  refreshAll();
  const sub = subscriptionsData.find((s) => s.id === id);
  showToast(
    sub && sub.status === 'paused'
      ? `Đã hủy gia hạn ${sub.name}`
      : 'Đã kích hoạt lại gói đăng ký',
    'ok'
  );
}

async function deleteSub(id) {
  applySnapshot(await call('DeleteSubscription', id));
  refreshAll();
  showToast('Đã xóa subscription.', 'ok');
}

// ------------------------------------------------------------------ modals

function openModal(id) {
  if (id === 'addSubModal') {
    populateSubGmailSelect();
    renderPresetPicker();
  }
  document.getElementById(id).classList.remove('hidden');
}

function closeModal(id) {
  if (id === 'quickVaultModal') quickVaultTarget = null;
  document.getElementById(id).classList.add('hidden');
}

async function openEditCredentialsModal() {
  const acc = findAccount(selectedGmail);
  if (!acc) return;
  document.getElementById('editEmailReadonly').value = acc.email;
  document.getElementById('editPassInput').value = acc.password;
  document.getElementById('editSecretInput').value = acc.secret2fa;
  document.getElementById('editRecoveryInput').value = acc.recoveryEmail;
  document.getElementById('editNoteInput').value = acc.note;
  openModal('editCredsModal');
}

async function saveEditedCredentials() {
  const acc = findAccount(selectedGmail);
  if (!acc) return;
  const updated = {
    ...acc,
    password: document.getElementById('editPassInput').value.trim() || acc.password,
    secret2fa:
      document.getElementById('editSecretInput').value.trim().replace(/\s/g, '').toUpperCase() ||
      acc.secret2fa,
    recoveryEmail: document.getElementById('editRecoveryInput').value.trim(),
    note: document.getElementById('editNoteInput').value.trim(),
  };
  applySnapshot(await call('SaveGmail', updated));
  closeModal('editCredsModal');
  refreshAll();
  showToast(`Đã cập nhật Mật khẩu & 2FA cho ${updated.email}`, 'ok');
}

async function openBackupCodesModal() {
  if (vaultLocked) {
    showToast('Mở khóa Két để xem mã dự phòng!', 'error');
    return;
  }
  const acc = findAccount(selectedGmail);
  if (!acc) return;
  document.getElementById('backupEmailTitle').innerText = acc.email;
  document.getElementById('backupCodesGrid').innerHTML = (acc.backupCodes || [])
    .map(
      (c, idx) => `
        <div onclick="copyString('${escJs(c)}', 'Đã copy mã dự phòng')" class="backup-code">
          <span>#${idx + 1}</span>
          <b>${esc(c)}</b>
        </div>`
    )
    .join('');
  openModal('backupCodesModal');
}

async function saveNewGmailAccount() {
  const email = document.getElementById('newGmailInput').value.trim();
  const pass = document.getElementById('newGmailPass').value.trim();
  const [tag, color] = document.getElementById('newGmailTag').value.split('|');
  const secret = document.getElementById('newGmail2faSecret').value.trim().replace(/\s/g, '').toUpperCase();
  const recovery = document.getElementById('newGmailRecovery').value.trim() || 'Chưa cài đặt';
  const phone = document.getElementById('newGmailPhone').value.trim() || 'Chưa liên kết';
  const note = document.getElementById('newGmailNote').value.trim() || 'Tài khoản Gmail cá nhân';

  if (!email || !email.includes('@')) {
    showToast('Vui lòng nhập Gmail hợp lệ!', 'error');
    return;
  }
  if (!pass) {
    showToast('Vui lòng nhập mật khẩu!', 'error');
    return;
  }
  if (gmailAccounts.some((a) => a.email.toLowerCase() === email.toLowerCase())) {
    showToast('Gmail này đã có trong danh sách!', 'error');
    return;
  }

  const account = {
    email,
    tag,
    color,
    initials: email.substring(0, 2).toUpperCase(),
    password: pass,
    secret2fa: secret || 'JBSWY3DPEHPK3PXP',
    recoveryEmail: recovery,
    phone,
    appPassword: 'Chưa tạo',
    note,
    backupCodes: [
      '1122 4433',
      '5566 8877',
      '9900 2211',
      '3344 6655',
      '7788 0099',
      '1234 8765',
      '4321 5678',
      '9876 5432',
    ],
  };

  applySnapshot(await call('SaveGmail', account));
  selectedGmail = email;
  document.getElementById('newGmailInput').value = '';
  document.getElementById('newGmailPass').value = '';
  document.getElementById('newGmail2faSecret').value = '';
  closeModal('addGmailModal');
  refreshAll();
  showToast(`Đã thêm ${email} kèm 2FA!`, 'ok');
}

function openAddSubForCurrentGmail() {
  populateSubGmailSelect();
  document.getElementById('subGmailSelect').value = selectedGmail;
  openModal('addSubModal');
}

function populateSubGmailSelect() {
  document.getElementById('subGmailSelect').innerHTML = gmailAccounts
    .map((a) => `<option value="${esc(a.email)}">${esc(a.email)} (${esc(a.tag)})</option>`)
    .join('');
}

function toggleSubCustomPassField() {
  document
    .getElementById('subCustomPassWrap')
    .classList.toggle('hidden', document.getElementById('subAuthModeSelect').value === 'sso');
}

async function saveNewSubscription() {
  const name = document.getElementById('subNameInput').value.trim();
  const price = parseFloat(document.getElementById('subPriceInput').value) || 0;
  const category = document.getElementById('subCatInput').value;
  const gmail = document.getElementById('subGmailSelect').value;
  const authMode = document.getElementById('subAuthModeSelect').value;
  const customPass = document.getElementById('subCustomPassInput').value.trim();
  const nextDate = document.getElementById('subDateInput').value || new Date().toISOString().slice(0, 10);
  const payment = document.getElementById('subCardInput').value.trim() || 'Ví MoMo';
  const wantIcon = imageServiceReady && (document.getElementById('subAiIconCheck')?.checked ?? true);

  if (!name || price <= 0) {
    showToast('Vui lòng nhập tên dịch vụ và giá!', 'error');
    return;
  }

  const sub = {
    id: 0,
    name,
    plan: document.getElementById('subPlanInput')?.value.trim() || `${category} • Cá nhân`,
    category,
    gmail,
    price,
    cycle: 'tháng',
    nextDate,
    nextText: '',
    urgency: 'normal',
    payment,
    receiptId: '',
    status: 'active',
    authMode,
    customPass: authMode === 'custom' ? customPass || 'Pass#12345' : '',
    iconBg: document.getElementById('subIconBgInput')?.value || 'bg-indigo-600',
    iconText:
      document.getElementById('subIconTextInput')?.value.trim() ||
      name.substring(0, 2).toUpperCase(),
    iconFile: '',
  };

  const snap = await call('SaveSubscription', sub);
  applySnapshot(snap);
  const created = subscriptionsData[0];

  selectedGmail = gmail;
  document.getElementById('subNameInput').value = '';
  document.getElementById('subPriceInput').value = '';
  closeModal('addSubModal');
  refreshAll();
  showToast(`Đã thêm ${name} vào ${gmail}`, 'ok');

  if (wantIcon && created) {
    await generateIconForSubscription(created.id, created.name);
  }
}

/**
 * Asks the backend to create an icon for one subscription and refreshes the
 * UI. The image service takes 15 to 30 seconds, so the button shows a
 * spinner while it works.
 * @param {number} id
 * @param {string} name
 */
async function generateIconForSubscription(id, name) {
  const button = document.querySelector(`[data-gen-icon="${id}"]`);
  const label = button ? button.textContent : '';
  if (button) {
    button.disabled = true;
    button.textContent = 'Đang tạo…';
  }
  showToast(`Đang tạo icon cho ${name} (mất khoảng 20-30 giây)…`);

  const result = await call('GenerateSubscriptionIcon', id);
  if (!result) {
    if (button) {
      button.disabled = false;
      button.textContent = label;
    }
    return;
  }

  iconCache.set(result.file, result.dataUri);
  applySnapshot(await call('GetState'));
  refreshAll();
  showToast(`Đã tạo icon cho ${name}!`, 'ok');
}

/**
 * Creates an AI avatar for the selected Gmail account.
 */
async function generateAvatarForAccount(email) {
  if (!email) return;
  showToast(`Đang tạo avatar cho ${email} (mất khoảng 20-30 giây)…`);
  const result = await call('GenerateAccountAvatar', email);
  if (!result) return;
  iconCache.set(result.file, result.dataUri);
  applySnapshot(await call('GetState'));
  refreshAll();
  showToast(`Đã tạo avatar cho ${email}!`, 'ok');
}

/**
 * Creates the artwork used for the desktop application icon.
 */
async function generateApplicationIcon() {
  showToast('Đang tạo icon ứng dụng (mất khoảng 20-30 giây)…');
  const result = await call('GenerateAppIcon');
  if (!result) return;
  iconCache.set(result.file, result.dataUri);
  refreshAll();
  showToast('Đã tạo icon ứng dụng!', 'ok');
}

// ------------------------------------------------------------------ presets

// The ready made subscription catalogue comes from the backend so both sides
// stay in sync with the same list.
let iconPresets = [];

/**
 * Loads the preset catalogue once and keeps it for the picker.
 * @returns {Promise<Array>}
 */
async function loadIconPresets() {
  if (iconPresets.length) return iconPresets;
  const presets = await call('GetIconPresets');
  iconPresets = Array.isArray(presets) ? presets : [];
  return iconPresets;
}

/**
 * Renders the searchable grid of presets inside the add-subscription modal.
 */
async function renderPresetPicker() {
  const grid = document.getElementById('presetGrid');
  if (!grid) return;
  const presets = await loadIconPresets();
  const query = (document.getElementById('presetSearchInput')?.value || '').toLowerCase().trim();

  const filtered = presets.filter(
    (p) =>
      !query ||
      p.name.toLowerCase().includes(query) ||
      p.plan.toLowerCase().includes(query) ||
      p.category.toLowerCase().includes(query)
  );

  if (filtered.length === 0) {
    grid.innerHTML = '<div class="empty">Không có dịch vụ nào khớp từ khoá.</div>';
    return;
  }

  grid.innerHTML = filtered.map((p) => presetCardHtml(p)).join('');
}

/**
 * Fills the add-subscription form with a preset.
 * @param {string} name
 */
function applyPreset(name) {
  const preset = iconPresets.find((p) => p.name === name);
  if (!preset) return;

  document.getElementById('subNameInput').value = preset.name;
  document.getElementById('subPriceInput').value = Number(preset.price).toFixed(2);
  document.getElementById('subCatInput').value = preset.category;
  document.getElementById('subCardInput').value = preset.payment;
  document.getElementById('subIconBgInput').value = preset.iconBg;
  document.getElementById('subIconTextInput').value = preset.iconText;

  const planEl = document.getElementById('subPlanInput');
  if (planEl) planEl.value = preset.plan;

  showToast(`Đã chọn mẫu ${preset.name}`, 'ok');
}

// ------------------------------------------------------------------ overview

function renderOverviewPills() {
  const container = document.getElementById('overviewGmailPills');
  const entries = [['all', `Tất cả Gmail (${gmailAccounts.length})`], ...gmailAccounts.map((a) => [a.email, a.email])];
  container.innerHTML = entries
    .map(
      ([value, label]) =>
        `<button type="button" class="filter-pill ${overviewGmailFilter === value ? 'is-active' : ''}" onclick="setOverviewGmail('${escJs(
          value
        )}')">${esc(label)}</button>`
    )
    .join('');
}

function setOverviewGmail(email) {
  overviewGmailFilter = email;
  renderOverviewPills();
  renderOverviewTable();
}

function renderOverviewTable() {
  const search = (document.getElementById('overviewSearchInput')?.value || '').toLowerCase().trim();
  const cat = document.getElementById('overviewCatSelect').value;
  const sort = document.getElementById('overviewSortSelect').value;

  let list = subscriptionsData.filter((s) => {
    if (overviewGmailFilter !== 'all' && s.gmail !== overviewGmailFilter) return false;
    if (cat !== 'all' && s.category !== cat) return false;
    if (!search) return true;
    return (
      s.name.toLowerCase().includes(search) ||
      s.gmail.toLowerCase().includes(search) ||
      (s.payment || '').toLowerCase().includes(search)
    );
  });

  if (sort === 'price-desc') list = [...list].sort((a, b) => b.price - a.price);
  else if (sort === 'name-asc') list = [...list].sort((a, b) => a.name.localeCompare(b.name));
  else list = [...list].sort((a, b) => new Date(a.nextDate) - new Date(b.nextDate));

  const body = document.getElementById('overviewTableBody');
  if (list.length === 0) {
    body.innerHTML =
      '<tr><td colspan="6"><div class="empty">Không có gói đăng ký nào khớp bộ lọc.<br /><strong>Thử nới lọc.</strong></div></td></tr>';
    return;
  }

  body.innerHTML = list
    .map((sub) => {
      const acc = findAccount(sub.gmail) || { color: 'blue' };
      const pillCls = pillClassMap[acc.color] || 'pill-blue';
      const renewalCls =
        sub.urgency === 'urgent'
          ? 'renewal-urgent'
          : sub.urgency === 'warning'
            ? 'renewal-warning'
            : 'renewal-normal';
      const pauseBtn =
        sub.status !== 'paused'
          ? `<button type="button" class="link-btn danger" onclick="toggleSubPause(${sub.id})">Hủy</button>`
          : `<button type="button" class="link-btn" style="color: var(--success-600)" onclick="toggleSubPause(${sub.id})">Mở lại</button>`;

      return `
          <tr class="${sub.status === 'paused' ? 'is-paused' : ''}">
            <td>
              <div class="cell-main">
                ${serviceIconHtml(sub)}
                <div style="min-width: 0">
                  <div class="cell-title">${esc(sub.name)}</div>
                  <div class="cell-sub">${esc(sub.plan)}</div>
                </div>
              </div>
            </td>
            <td>
              <div class="row" style="gap: 8px; flex-wrap: wrap">
                <span class="tag-pill ${pillCls} mono">${esc(sub.gmail)}</span>
                <button type="button" class="link-btn" onclick="openQuickVault(${sub.id})"><svg width="14" height="14"><use href="#i-key" /></svg> Pass &amp; 2FA</button>
              </div>
            </td>
            <td class="text-right">
              <div class="cell-title tabular-nums">$${sub.price.toFixed(2)}<span
                  class="cell-sub"
                  style="font-weight: 400"
                  >/${esc(sub.cycle)}</span
                ></div>
              <div class="cell-sub tabular-nums">~${formatVnd(sub.price)}</div>
            </td>
            <td>
              <div><span class="renewal-badge ${renewalCls}">${esc(sub.nextText)}</span></div>
              <div class="cell-sub">${esc(sub.nextDate)}</div>
            </td>
            <td>
              <div class="cell-title">${esc(sub.payment)}</div>
              <button type="button" class="link-btn" onclick="previewReceiptModal('${escJs(
                sub.name
              )}', '$${sub.price.toFixed(2)} (~${formatVnd(sub.price)})', '${escJs(
                sub.receiptId
              )}', '${escJs(sub.gmail)}', '${escJs(sub.payment)}', '${escJs(
                sub.nextDate
              )}')">#${esc(sub.receiptId)}</button>
            </td>
            <td class="text-right">
              <div class="actions" style="justify-content: flex-end">
                ${genIconButtonHtml(sub)}
                ${pauseBtn}
              </div>
            </td>
          </tr>
        `;
    })
    .join('');
}

function renderPersonalBreakdown() {
  const container = document.getElementById('personalSpendBreakdown');
  const active = activeSubscriptions();
  const total = active.reduce((sum, s) => sum + s.price, 0) || 1;

  const sorted = gmailAccounts
    .map((acc) => {
      const spend = active.filter((s) => s.gmail === acc.email).reduce((sum, s) => sum + s.price, 0);
      return { ...acc, spend, pct: (spend / total) * 100 };
    })
    .sort((a, b) => b.spend - a.spend);

  container.innerHTML = sorted
    .map(
      (acc) => `
        <div class="breakdown-item">
          <header>
            <span class="row" style="gap: 8px; min-width: 0">
              <span class="avatar ${avatarClass(acc.color)}" style="width: 26px; height: 26px; flex-basis: 26px; font-size: 10px" data-icon-file="${esc(
                acc.iconFile || ''
              )}">${esc(acc.initials)}</span>
              <span class="truncate">${esc(acc.email)}</span>
              <span class="text-faint">(${esc(acc.tag)})</span>
            </span>
            <b class="tabular-nums">$${acc.spend.toFixed(2)} <span class="text-faint" style="font-weight: 400">(${formatVnd(
              acc.spend
            )})</span></b>
          </header>
          <div class="progress">
            <span class="${barClassMap[acc.color] || 'bar-brand'}" style="width: ${acc.pct.toFixed(1)}%"></span>
          </div>
        </div>`
    )
    .join('');
}

function renderPersonalAlerts() {
  const box = document.getElementById('personalAlertsBox');
  const items = expiringSubscriptions();
  if (items.length === 0) {
    box.innerHTML =
      '<div class="empty">Chưa có gói nào sắp hết hạn trong vài ngày tới.<br /><strong>Tất cả ổn.</strong></div>';
    return;
  }
  box.innerHTML = items
    .map(
      (s) => `
        <div class="alert-row ${s.urgency === 'urgent' ? 'urgent' : 'warning'}">
          <div style="min-width: 0">
            <b>${esc(s.name)} — ${esc(s.nextText)}</b>
            <span>Gmail: <code class="mono">${esc(s.gmail)}</code> · $${s.price.toFixed(2)} (${esc(
              s.payment
            )})</span>
          </div>
          <button type="button" class="btn btn-primary btn-sm" style="flex: 0 0 auto" onclick="switchMainView('vault'); selectVaultAccount('${escJs(
            s.gmail
          )}')">Xem Mail</button>
        </div>`
    )
    .join('');
}

// ------------------------------------------------------------------ quick vault

async function openQuickVault(subId) {
  if (vaultLocked) {
    showToast('Két đang khóa!', 'error');
    return;
  }
  const sub = subscriptionsData.find((s) => s.id === subId);
  const acc = sub && findAccount(sub.gmail);
  if (!acc) return;

  quickVaultTarget = sub;
  document.getElementById('qvTitle').innerText = `Đăng nhập nhanh: ${sub.name}`;
  document.getElementById('qvEmail').innerText = acc.email;
  document.getElementById('qvPass').innerText = acc.password;

  const customBox = document.getElementById('qvCustomPassBox');
  if (sub.authMode === 'custom' && sub.customPass) {
    customBox.classList.remove('hidden');
    document.getElementById('qvCustomPass').innerText = sub.customPass;
  } else {
    customBox.classList.add('hidden');
  }
  tickTotp();
  openModal('quickVaultModal');
}

async function copyQuickVaultPass() {
  if (!quickVaultTarget) return;
  await copyAccountSecret(quickVaultTarget.gmail, 'password', 'Đã sao chép mật khẩu Gmail!');
}

async function copyQuickVaultTotp() {
  if (!quickVaultTarget) return;
  const res = await call('GetTotp', quickVaultTarget.gmail);
  if (!res) return;
  await copyString(res.code, `Đã sao chép mã 2FA: ${res.code}`);
}

async function copyQuickCustomPass() {
  if (!quickVaultTarget) return;
  await copyCustomPass(quickVaultTarget.id, quickVaultTarget.name);
}

function previewReceiptModal(name, price, receiptId, gmail, payment, timestamp) {
  document.getElementById('modalServiceName').innerText = `Biên Lai: ${name}`;
  document.getElementById('modalInvoiceId').innerText = `Mã giao dịch: #${receiptId}`;
  document.getElementById('modalTargetGmail').innerText = gmail;
  document.getElementById('modalPaymentMethod').innerText = payment;
  document.getElementById('modalTimestamp').innerText = timestamp;
  document.getElementById('modalTotalPrice').innerText = price;
  openModal('receiptModal');
}

// ------------------------------------------------------------------ backup

async function exportJsonBackup() {
  const path = await call('SaveBackupDialog');
  if (!path) return;
  const result = await call('ExportBackup', path);
  if (result === null) return;
  showToast(`Đã sao lưu tại: ${result}`, 'ok');
}

async function importJsonBackup() {
  const path = await call('OpenBackupDialog');
  if (!path) return;
  applySnapshot(await call('ImportBackup', path));
  selectedGmail = gmailAccounts[0]?.email || '';
  closeModal('backupDataModal');
  refreshAll();
  showToast('Khôi phục dữ liệu thành công!', 'ok');
}

async function clearAllData() {
  if (!confirm('Xóa toàn bộ tài khoản Gmail và gói đăng ký đang lưu?')) return;
  applySnapshot(await call('ClearAllData'));
  selectedGmail = gmailAccounts[0]?.email || '';
  closeModal('backupDataModal');
  refreshAll();
  showToast('Đã xóa toàn bộ dữ liệu.', 'ok');
}

// ------------------------------------------------------------------ chrome helpers

/** Shows a transient message in the bottom-right corner. */
function showToast(msg, kind = '') {
  const host = document.getElementById('toastHost');
  if (!host) return;
  const toast = document.createElement('div');
  toast.className = `toast ${kind}`.trim();
  toast.innerText = msg;
  host.appendChild(toast);
  setTimeout(() => toast.remove(), 2800);
}

/** Updates the sidebar badges and the header counters. */
function renderShellChrome() {
  const active = activeSubscriptions();
  const expiring = expiringSubscriptions();

  document.getElementById('navAccountCount').innerText = gmailAccounts.length;
  document.getElementById('navSubCount').innerText = active.length;
  document.getElementById('navAlertCount').innerText = expiring.length;

  const badge = document.getElementById('sidebarVaultBadge');
  badge.className = `badge ${vaultLocked ? 'locked' : 'active'}`;
  badge.innerText = vaultLocked ? 'khóa' : 'mở';
  document.getElementById('sidebarVaultHint').innerText = vaultLocked
    ? `${gmailAccounts.length} tài khoản — bí mật đang được che`
    : `${gmailAccounts.length} tài khoản — sẵn sàng hiển thị`;
  document.getElementById('userLockLabel').innerText = vaultLocked ? 'Mở khóa két' : 'Khóa két';

  const systemBadge = document.getElementById('systemVaultBadge');
  if (systemBadge) {
    systemBadge.className = `badge ${vaultLocked ? 'locked' : 'active'}`;
    systemBadge.innerText = vaultLocked ? 'đang khóa' : 'đang mở';
  }

  const alertCount = document.getElementById('alertCount');
  alertCount.innerText = expiring.length;
  alertCount.classList.toggle('hidden', expiring.length === 0);
  document.getElementById('alertsSummary').innerText =
    expiring.length === 0 ? 'Không có bất thường' : `${expiring.length} gói cần chú ý`;

  const alertsList = document.getElementById('alertsList');
  alertsList.innerHTML =
    expiring.length === 0
      ? '<div class="dropdown-item" style="cursor: default">Không có gói nào sắp gia hạn</div>'
      : expiring
          .map(
            (s) => `<button type="button" class="dropdown-item" onclick="switchMainView('vault'); selectVaultAccount('${escJs(
              s.gmail
            )}')">
              <svg><use href="#i-alert" /></svg> ${esc(s.name)} — ${esc(s.nextText)}
            </button>`
          )
          .join('');

  const systemAccountCount = document.getElementById('systemAccountCount');
  if (systemAccountCount) systemAccountCount.innerText = gmailAccounts.length;
  const systemSubCount = document.getElementById('systemSubCount');
  if (systemSubCount) systemSubCount.innerText = subscriptionsData.length;
}

/** Updates the metric cards on the overview page. */
function renderMetrics() {
  const active = activeSubscriptions();
  const total = totalSpend();
  const expiring = expiringSubscriptions();

  document.getElementById('metricGmails').innerText = gmailAccounts.length;
  document.getElementById('metricSubs').innerText = active.length;
  document.getElementById('metricCost').innerText = `$${total.toFixed(2)}`;
  document.getElementById('metricCostVnd').innerText = `~${formatVnd(total)}`;
  document.getElementById('metricExpiring').innerText = expiring.length;

  const paused = subscriptionsData.length - active.length;
  document.getElementById('metricSubsTrend').innerText =
    paused > 0 ? `${paused} đã hủy` : 'tất cả';
  document.getElementById('metricGmailsTrend').innerText = 'cá nhân';
  document.getElementById('metricCostTrend').innerText = 'USD';
  document.getElementById('metricExpiringTrend').innerText = 'trong 5 ngày';
}

/** Applies the global search term to both the vault list and the overview. */
function applyGlobalQuery(query) {
  globalQuery = query;
  const vaultSearch = document.getElementById('vaultSearchGmail');
  const overviewSearch = document.getElementById('overviewSearchInput');
  if (vaultSearch && vaultSearch.value !== query) vaultSearch.value = query;
  if (overviewSearch && overviewSearch.value !== query) overviewSearch.value = query;
  renderVaultAccountList();
  renderOverviewTable();
}

function refreshAll() {
  renderShellChrome();
  renderMetrics();
  renderVaultTagPills();
  renderVaultAccountList();
  renderVaultDetail();
  renderOverviewPills();
  renderOverviewTable();
  renderPersonalBreakdown();
  renderPersonalAlerts();
  tickTotp();
  hydrateIcons();
}

// ------------------------------------------------------------------ events

function setupEvents() {
  document.getElementById('sidebarToggle').addEventListener('click', toggleSidebar);
  document.getElementById('themeToggle').addEventListener('click', toggleTheme);

  document.getElementById('sidebarOverlay').addEventListener('click', closeSidebar);

  document.getElementById('globalSearch').addEventListener('input', (event) => {
    applyGlobalQuery(event.target.value);
  });

  document.querySelectorAll('.menu-item[data-nav]').forEach((item) => {
    item.addEventListener('click', (event) => {
      event.preventDefault();
      const nav = item.dataset.nav;
      switchMainView(nav === 'alerts' ? 'overview' : nav);
      history.replaceState(null, '', `#/${activePage}`);
    });
  });

  document.addEventListener('click', (event) => {
    const trigger = event.target.closest('.dropdown > button');
    if (trigger) {
      const parent = trigger.parentElement;
      const willOpen = !parent.classList.contains('is-open');
      document.querySelectorAll('.dropdown.is-open').forEach((d) => d.classList.remove('is-open'));
      parent.classList.toggle('is-open', willOpen);
      trigger.setAttribute('aria-expanded', String(willOpen));
      return;
    }
    if (!event.target.closest('.dropdown')) {
      document.querySelectorAll('.dropdown.is-open').forEach((d) => d.classList.remove('is-open'));
    }

    const actionEl = event.target.closest('[data-action]');
    if (!actionEl) return;
    const action = actionEl.dataset.action;
    if (action === 'refresh-all') {
      reload();
      showToast('Đã làm mới dữ liệu!', 'ok');
    } else if (action === 'toggle-lock') {
      toggleVaultLock();
    } else if (action === 'open-backup') {
      openModal('backupDataModal');
    }
  });

  document.addEventListener('keydown', (event) => {
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
      event.preventDefault();
      const search = document.getElementById('globalSearch');
      search.focus();
      search.select();
    } else if (event.key === 'Escape') {
      closeSidebar();
      document.querySelectorAll('.dropdown.is-open').forEach((d) => d.classList.remove('is-open'));
      const search = document.getElementById('globalSearch');
      if (document.activeElement === search && search.value) {
        search.value = '';
        applyGlobalQuery('');
      }
      document.querySelectorAll('.auth-gate:not(.hidden)').forEach((modal) => {
        if (modal.id) closeModal(modal.id);
      });
    }
  });

  window.addEventListener('hashchange', () => {
    const page = currentPage();
    if (page !== activePage) switchMainView(page);
  });

  window.addEventListener('resize', () => {
    if (window.innerWidth >= 1280) closeSidebar();
  });
}

// Expose the handlers used by inline onclick attributes.
window.selectVaultAccount = selectVaultAccount;
window.switchMainView = switchMainView;
window.toggleTheme = toggleTheme;
window.toggleVaultLock = toggleVaultLock;
window.setVaultTagFilter = setVaultTagFilter;
window.toggleDetailPassword = toggleDetailPassword;
window.copySelectedPassword = copySelectedPassword;
window.copySelectedTotp = copySelectedTotp;
window.copySecretKey = copySecretKey;
window.copyAppPassword = copyAppPassword;
window.copyAllBackupCodes = copyAllBackupCodes;
window.copyString = copyString;
window.copyTextById = copyTextById;
window.copyCustomPass = copyCustomPass;
window.deleteSelectedGmail = deleteSelectedGmail;
window.toggleSubPause = toggleSubPause;
window.deleteSub = deleteSub;
window.openEditCredentialsModal = openEditCredentialsModal;
window.saveEditedCredentials = saveEditedCredentials;
window.openBackupCodesModal = openBackupCodesModal;
window.saveNewGmailAccount = saveNewGmailAccount;
window.openAddSubForCurrentGmail = openAddSubForCurrentGmail;
window.toggleSubCustomPassField = toggleSubCustomPassField;
window.saveNewSubscription = saveNewSubscription;
window.generateIconForSubscription = generateIconForSubscription;
window.generateAvatarForAccount = generateAvatarForAccount;
window.generateApplicationIcon = generateApplicationIcon;
window.renderPresetPicker = renderPresetPicker;
window.applyPreset = applyPreset;
window.setOverviewGmail = setOverviewGmail;
window.openQuickVault = openQuickVault;
window.copyQuickVaultPass = copyQuickVaultPass;
window.copyQuickVaultTotp = copyQuickVaultTotp;
window.copyQuickCustomPass = copyQuickCustomPass;
window.previewReceiptModal = previewReceiptModal;
window.exportJsonBackup = exportJsonBackup;
window.importJsonBackup = importJsonBackup;
window.clearAllData = clearAllData;
window.openModal = openModal;
window.closeModal = closeModal;

setupEvents();
switchMainView(currentPage());
history.replaceState(null, '', `#/${activePage}`);

/**
 * Asks the backend which integrations are enabled on this machine. The image
 * service is optional, so the UI only offers the matching action when one is
 * actually configured.
 * @returns {Promise<void>}
 */
async function loadIntegrations() {
  const list = await call('GetIntegrations');
  const items = Array.isArray(list) ? list : [];
  imageServiceReady = items.some((it) => it && it.type === 'image' && it.enabled);

  for (const id of ['btnGenAvatar', 'btnGenAppIcon', 'subAiIconRow']) {
    const el = document.getElementById(id);
    if (el) el.classList.toggle('hidden', !imageServiceReady);
  }
}

loadIntegrations().then(refreshAll);

reload().then(() => {
  call('DataPath').then((p) => {
    const el = document.getElementById('systemDataPath');
    if (el && p) el.innerText = p;
  });
});
