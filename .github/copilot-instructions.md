# Copilot Instructions — SubMail Personal (Gmail Subscription Manager)

## Tổng quan dự án

Đây là **ứng dụng desktop Go + Wails v2**. Bản MVP nằm trong thư mục
[`desktop/`](../desktop):

```text
desktop/
├── main.go                  # Entry point + //go:embed all:frontend/dist
├── app.go                   # Bound methods (state, CRUD, lock, TOTP, backup, dialogs)
├── icons.go                 # AI icon generation (subscription / avatar / app icon)
├── cmd/genicon/             # Tool sinh icon ứng dụng + file .ico
├── internal/model/          # GmailAccount, Subscription, State
├── internal/totp/           # Base32 + HMAC-SHA1 TOTP (RFC 6238)
├── internal/store/          # Đọc/ghi JSON atomic + thư mục ảnh
├── internal/icon/           # Xử lý PNG/ICO + prompt dựng sẵn
├── internal/integrations/   # Cấu hình & adapter gọi dịch vụ tuỳ chọn
└── frontend/                # Vite + vanilla JS (không framework)
```

[`index.html`](../index.html) ở thư mục gốc là **prototype web cũ**, được giữ lại làm tham chiếu
giao diện — **không phải bản chạy chính**. Sửa logic ở `desktop/`, không sửa ở đó.

Chức năng: két chứa nhiều tài khoản **Gmail cá nhân** (mật khẩu, email khôi phục, SĐT, App
Password, 8 mã dự phòng 2FA) kèm **mã TOTP 6 số sống thật theo RFC 6238**, cộng với bộ theo
dõi các **gói đăng ký (subscriptions)** đăng ký dưới từng Gmail — chi phí, ngày gia hạn, thẻ
/ví thanh toán, cảnh báo trial/sắp gia hạn, và (tuỳ chọn) icon tạo bằng AI.

Ngôn ngữ toàn bộ giao diện và tài liệu trong repo là **tiếng Việt**. Giữ nguyên điều này.

## Lệnh build, chạy và kiểm thử

```text
cd desktop
wails dev                    # chạy dev (hot reload)
wails build                  # build .exe -> desktop/build/bin/submail-personal.exe
go test ./...                # test TOTP + store + app + integrations
cd frontend && npm test      # test esc() chống XSS (node --test)
```

Yêu cầu toolchain: **Go 1.26+**, **Node.js 20+**, **Wails CLI v2**, gcc (cgo) cho Windows.

**Không thêm** framework test Go (testify, ginkgo) hay test runner JS (jest, vitest). Kiểm thử
dùng `go test` và `node --test` thuần — đủ cho toàn bộ logic ở đây.

## Kiến trúc: Go giữ bí mật, frontend chỉ hiển thị

Đây là quyết định kiến trúc quan trọng nhất — **không phá vỡ nó**:

- **Bí mật chỉ nằm ở tầng Go.** Frontend **không bao giờ** nhận `password` / `secret2fa` /
  `backupCodes` / `customPass` khi két đang khóa.
- `App.GetState()` trả về mask (`••••••••••••••`) thay vì giá trị thật khi `locked == true`.
- `App.RevealSecret(email, field)` và `App.RevealCustomPass(id)` **từ chối** khi két khóa.
  Frontend gọi các hàm này, không đọc bí mật từ payload sẵn có.
- Hệ quả: muốn vá lỗi "lộ mật khẩu" thì sửa **một chỗ ở Go**, không vá từng nút copy ở JS.

### Mô hình dữ liệu (`internal/model`)

- **`GmailAccount`**: `email` (khóa chính, duy nhất), `tag`, `color`, `initials`, `password`,
  `secret2fa` (Base32), `recoveryEmail`, `phone`, `appPassword`, `note`, `backupCodes[]`.
- **`Subscription`**: `id` (int64), `name`, `plan`, `category`, `gmail`, `price` (USD), `cycle`,
  `nextDate` (ISO), `payment`, `receiptId`, `status`, `authMode`, `customPass`, `iconBg`,
  `iconText`.
- **`State`**: `{ gmailAccounts, subscriptionsData }` — khớp đúng tên key của prototype web cũ.

Giá trị hợp lệ (có hằng số + hàm kiểm tra trong `model`):

- `status`: `active` | `trial` | `paused` — **`paused` bị loại khỏi mọi phép tính chi phí và cảnh
  báo** (mọi nơi lọc `s.status !== 'paused'`).
- `authMode`: `sso` (dùng chung pass + 2FA của Gmail) | `custom` (mật khẩu riêng trong
  `customPass`).
- `category`: `AI` | `Media` | `Dev` | `Storage` | `Shopping`.
- `color`: `blue` | `purple` | `amber` | `emerald` | `rose`.

`SaveGmail` và `SaveSubscription` **từ chối** giá trị ngoài tập hợp trên — giữ nguyên, đừng nới
lỏng mà không đổi **cả hai** map màu (`pillClassMap`, `barClassMap` ở
`frontend/src/main.js`). Thiếu một map sẽ ra `undefined` in ra HTML.

Ngoài ra `iconBg` của subscription vẫn lưu **chuỗi class Tailwind cũ** (`bg-emerald-600`,
`bg-sky-600`, …) trong `data.json` — đừng đổi định dạng lưu trữ, hãy dịch sang class mới ở
tầng hiển thị qua map `iconClassMap` trong `frontend/src/main.js`.

## State & lưu trữ (`internal/store`)

- Đường dẫn: `os.UserConfigDir()/SubMailPersonal/data.json` → trên Windows là
  `%AppData%\SubMailPersonal\data.json`.
- `Load()`: file hỏng hoặc thiếu → trả về **state rỗng** (`model.NewState()`), **không** trả lỗi.
- `Save()`: ghi **atomic** — ghi file tạm trong cùng thư mục rồi `os.Rename` sang đích, nên
  không để lại file nửa vời khi app bị tắt đột ngột.
- `Export()` / `Import()`: dùng cho file sao lưu `.json` do người dùng chọn qua hộp thoại native.
- `IconsDir()` / `EnsureIconsDir()`: thư mục `icons/` cạnh file dữ liệu, chứa PNG do dịch vụ
  tạo ảnh sinh ra.

### App KHÔNG đóng gói dữ liệu mẫu

Repo này là **open source**: không được đưa bất kỳ dữ liệu trông như thật nào vào mã nguồn
(email `@gmail.com` thật dạng, mật khẩu, secret Base32, App Password, SĐT). App mở lên
**trống**, người dùng tự thêm tài khoản.

- Không có `DemoState()` trong production code. Fixture cho test nằm **trong file test**.
- `App.ClearAllData()` xoá sạch két (thay cho `ResetDemoData` cũ).
- Nếu cần dữ liệu minh hoạ, dùng giá trị **lộ liễu là giả**: `vi-du-1@example.com`,
  `MAT-KHAU-MINH-HOA-1`, `DEMO2FAKEY00000001`.

### Ngày gia hạn được tính ở Go

`nextText` và `urgency` **không** lưu trong file. Go tính lại mỗi lần render
(`renewalText` / `renewalUrgency` trong `app.go`) từ `nextDate`:
`Hôm nay` / `Ngày mai (24h)` / `Còn N ngày` / `Đã quá hạn`, và
`urgent` (≤ 1 ngày hoặc trial) / `warning` (≤ 5 ngày) / `normal`.

Đây là **thay đổi có chủ đích** so với prototype web (ở đó `nextText` là chuỗi tĩnh lưu sẵn và
lệch dần theo thời gian). Đừng chuyển ngược về chuỗi tĩnh.

## TOTP (`internal/totp`)

- Base32 tự giải mã (bỏ khoảng trắng, `-`, `=`, chấp nhận chữ thường) → `crypto/hmac` +
  `crypto/sha1` → RFC 6238, **6 chữ số**.
- `Code(secret, t)` trả về chuỗi 6 số đã zero-pad; `Remaining(t)` trả về số giây còn lại.
- **Đã có test vector chính thức RFC 6238** trong `totp_test.go` (T=59 → `287082`,
  T=1111111109 → `081804`, …). Không sửa file test này để "cho qua" — nó là hàng rào.
- Frontend chỉ gọi `App.GetTotp(email)`. **Không** cài lại TOTP bằng `crypto.subtle` ở JS —
  bản web cũ cần secure context nên rơi về hash giả khi mở `file://`.
- Mã hiển thị có khoảng trắng ở giữa (`"482 915"`); bản copy đã bỏ khoảng trắng ở Go.

## Rendering (frontend)

Vanilla JS + CSS gốc trong `frontend/src/style.css`. **Không** dùng Tailwind CDN, không framework,
không virtual DOM. Giao diện bám theo hệ TailAdmin v2 (font Outfit, brand `#465fff`, radius
16px, sidebar cố định 290px + topbar có ô tìm kiếm `Ctrl K`).

- Mỗi vùng danh sách/bảng đổ bằng `element.innerHTML = array.map(...).join('')`.
- Hành động gắn bằng **inline `onclick`**. Vì dữ liệu nội suy vào chuỗi HTML, **bắt buộc**:
  - mọi **text node** dùng `esc(value)`;
  - mọi **chuỗi trong `onclick`** dùng `escJs(value)` (thoát `\`, `'`, `"`, `&`, `<`/`>`).
    `escJs` **phải** thoát cả `"` vì handler nằm trong thuộc tính HTML gấp kép — bỏ sót `"` là
    payload từ file sao lưu có thể chèn thêm thuộc tính `onmouseover=` (đã từng xảy ra).
  - Bỏ qua bước này là payload trong file sao lưu chạy được JavaScript (đã từng xảy ra).
- `esc()` / `escJs()` ở `frontend/src/escape.js`, có test ở `escape.test.mjs`.
- Copy dùng `App.CopyToClipboard` (clipboard native qua Wails), không dùng
  `navigator.clipboard`.
- Ba trang (`vault` | `overview` | `system`): `switchMainView(view)` bật `.page.is-active` và
  đồng bộ `.menu-item.is-active` + tiêu đề topbar.
- Modal: `openModal(id)` / `closeModal(id)` thao tác `classList`; `closeModal('quickVaultModal')`
  phải reset `quickVaultTarget = null`.
- Toast: `showToast(msg, kind)` — 2800ms ở góc dưới phải, `kind` là `''` | `'ok'` | `'error'`.

### Quy ước giao diện

- Chuỗi hiển thị, label, toast: **tiếng Việt**. Comment trong mã (Go và JS): **tiếng Anh**.
- Số tiền: luôn hiện **cả USD lẫn VNĐ** qua `formatVnd(usd)` (`vndRate = 25400`, locale `vi-VN`).
- Mọi con số cần thẳng hàng dùng class `tabular-nums`; mã 2FA/mật khẩu/secret dùng `font-mono`.
- Màu nhấn chính là **brand** (`--brand-500: #465fff`); `purple` chỉ dùng cho nhánh AI.
- **Mọi màu đi qua CSS custom property**, không hard-code trong markup. Token nền:
  `var(--surface)`, `var(--surface-muted)`, `var(--surface-inset)`, `var(--page-bg)`,
  `var(--border)`, `var(--border-soft)`, `var(--text-strong)`, `var(--text)`,
  `var(--text-muted)`, `var(--text-faint)`. Thêm biến màu mới thì khai báo **cả hai** block
  `:root` và `html.dark` trong `frontend/src/style.css`, nếu không sẽ vỡ ở một trong hai chế độ.
- Font **Outfit** được bundle tại `frontend/src/assets/fonts/` (`outfit-latin.woff2` +
  `outfit-latin-ext.woff2`). Phải giữ subset `latin-ext` — dấu tiếng Việt nằm ngoài subset
  `latin`.
- Giá trị `tag` là **chuỗi tiếng Việt nguyên văn** và được dùng làm điều kiện lọc — đổi nhãn
  nghĩa là đổi cả logic lọc.

## Luồng cập nhật dữ liệu

Mọi thao tác thay đổi dữ liệu ở Go đi theo **đúng thứ tự này**:

```text
sửa state  →  a.persist()  →  trả Snapshot  →  frontend applySnapshot() + refreshAll()
```

- Bỏ `persist()` nghĩa là mất dữ liệu khi đóng app.
- Bỏ `refreshAll()` ở frontend nghĩa là UI hiển thị cũ.
- Thêm mới dùng `append([]T{new}, old...)` (mục mới lên đầu); xoá dùng `.filter()`.
- Xoá một Gmail phải xoá **cả** các subscription tham chiếu tới nó (`DeleteGmail`).
- Xoá được tài khoản Gmail **cuối cùng** — két phải xoá được sạch. Dùng `ClearAllData` để
  xoá toàn bộ.

## Tích hợp tuỳ chọn (`internal/integrations`)

Dịch vụ tạo ảnh là **tuỳ chọn**, **mặc định tắt**, và cấu hình nằm **ngoài repo**:

- File cấu hình: `%AppData%\SubMailPersonal\integrations.json`, hoặc trỏ bằng biến môi
  trường `SUBMAIL_INTEGRATIONS_CONFIG`.
- Mẫu schema (không chứa bí mật thật): `desktop/integrations.example.json`.
- `App.GetIntegrations()` chỉ trả `id`, `name`, `type`, `enabled` — **không bao giờ** trả
  `apiKey` hay `baseUrl` lên frontend.
- Không có file cấu hình → app **không** gọi mạng và UI **không** hiện nút tạo ảnh.
- Frontend quyết định hiện nút qua `loadIntegrations()` → cờ `imageServiceReady`.
- `internal/icon` chỉ lo **xử lý ảnh** (PNG/ICO, resize, prompt). Gọi HTTP thuộc về
  `internal/integrations`.

## Bảo mật — cần hiểu trước khi "sửa"

- **Khóa Két (`SetLocked`) chỉ là che hiển thị ở tầng giao diện.** Dữ liệu gốc vẫn nằm trong
  file JSON **dạng rõ** dưới `%AppData%`. Không mô tả tính năng này là mã hoá.
- File backup `.json` chứa **toàn bộ bí mật ở dạng rõ**. Cảnh báo người dùng khi hướng dẫn
  dùng export.
- **Đã sửa:** các đường copy từng không kiểm tra trạng thái khóa (copy `customPass` trong bảng,
  copy mã dự phòng, copy trong modal Quick Vault). Nay **mọi** đường copy đi qua
  `copyAccountSecret` / `copyCustomPass` và bị Go từ chối khi khóa — không nhân bản lại lỗi cũ.
- `esc()` / `escJs()` là hàng rào XSS — không được bỏ khi thêm chỗ hiển thị dữ liệu mới.
- **Két khóa ⇒ không được ghi đè bí mật.** Khi `a.locked == true`, payload frontend chỉ chứa
  mask (`••••••••••••••`), nên `SaveGmail` / `SaveSubscription` phải **giữ nguyên** secret đã
  lưu thay vì ghi giá trị từ payload — nếu không sẽ **xoá sạch mật khẩu thật**. Có test hồi
  quy ở `icons_test.go`.

## Ảnh icon — cần hiểu trước khi "sửa"

- Ảnh AI lưu ở `%AppData%\SubMailPersonal\icons\`, tên file luôn là **tên đơn giản**;
  `internal/icon.ValidName` chặn `..` và dấu phân tách đường dẫn.
- Frontend nhận ảnh dưới dạng **data-URI** qua `App.GetIconData(file)`. **Đừng** phục vụ
  ảnh qua HTTP: trong `wails dev`, Vite chạy `appType:"spa"` nên `htmlFallbackMiddleware`
  ghi đè mọi URL lạ về `/index.html` ⇒ ảnh hỏng. Data-URI chạy giống nhau ở dev và prod.
- Lệnh gọi HTTP tạo ảnh mất 15-30 giây ⇒ **không được giữ `a.mu`** trong lúc gọi, nếu
  không toàn bộ UI bị treo. Chỉ khoá lúc ghi state + `persist()`.
- Dịch vụ ảnh **không nhất quán về định dạng**: URL thường kết thúc bằng `.png` nhưng nội
  dung có thể là JPEG. `icon.DecodePNG` nhận cả hai — đừng siết lại chỉ PNG.
- Prompt sinh ảnh mô tả **màu sắc và hình khối**, không yêu cầu tái tạo logo thương hiệu.

## Cạm bẫt cần biết

- `loadState()` / `saveState()` ở prototype web cũ nuốt lỗi im lặng — ở bản Go, `Load()` trả
  về seed thay vì panic; `Save()` trả lỗi để hiển thị.
- `os.UserConfigDir()` có thể lỗi → `startup` fallback sang `submail-data.json` cạnh app.
- `frontend/dist/` phải tồn tại (có `.gitkeep`) để `//go:embed all:frontend/dist` không làm
  hỏng `go test ./...` khi chưa build.
- `desktop/frontend/node_modules/` và `desktop/build/bin/` không commit.
- File `index.html` (prototype cũ) là **UTF-8 không BOM**, xuống dòng **LF**. Giữ nguyên khi
  chỉnh sửa.

## Ngoài phạm vi MVP

Đồng bộ Gmail qua IMAP, tự phân tích hoá đơn từ email, kho biên lai lịch sử, nhật ký kiểm toán,
Rule Automation, AI tối ưu chi phí, mã hoá at-rest bằng mật khẩu chủ, đồng bộ cloud, nhiều
người dùng. Các ảnh trong `mockups/` mô tả một sản phẩm rộng hơn nhiều (sidebar nav,
Analytics, Receipts Archive) — chưa được dựng.
