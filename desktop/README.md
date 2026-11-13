# SubMail Personal — Desktop (Go + Wails)

Bản desktop của SubMail Personal: quản lý nhiều tài khoản **Gmail cá nhân** (mật khẩu +
2FA TOTP sống + 8 mã dự phòng) kèm các **gói đăng ký** theo từng hộp thư.

Tài liệu đầy đủ (tính năng, bảo mật, tích hợp, kiểm thử): [`../README.md`](../README.md).

## Chạy

```bash
wails dev      # chế độ phát triển, hot reload
wails build    # build -> build/bin/submail-personal.exe
```

## Kiểm thử

```bash
go test ./...                    # TOTP RFC 6238, store, app, icon, integrations
cd frontend && npm test          # esc()/escJs() chống XSS + markup icon/preset
```

Kiểm thử với dịch vụ tạo ảnh thật (bỏ qua nếu không đặt biến môi trường):

```bash
SUBMAIL_IMAGE_INTEGRATION=1 go test -run TestImageIntegration ./...
```

## Bố cục

| Đường dẫn | Vai trò |
|---|---|
| `main.go` | Entry point, `//go:embed all:frontend/dist` |
| `app.go` | Bound methods: state, CRUD, lock, TOTP, backup, hộp thoại file |
| `icons.go` | Tạo icon AI cho gói / avatar / ứng dụng |
| `cmd/genicon/` | Tool sinh icon ứng dụng + file `.ico` cho Wails |
| `internal/model/` | Kiểu dữ liệu + tập giá trị hợp lệ |
| `internal/totp/` | Base32 + HMAC-SHA1 TOTP (RFC 6238) |
| `internal/store/` | Đọc/ghi JSON atomic + thư mục ảnh |
| `internal/icon/` | Xử lý PNG/ICO + prompt dựng sẵn |
| `internal/integrations/` | Cấu hình & adapter gọi dịch vụ tuỳ chọn |
| `frontend/` | Vite + vanilla JS; `src/escape.js` là hàng rào chống XSS |

## Ảnh icon

Ảnh do AI sinh nằm trong `%AppData%\SubMailPersonal\icons\`. Frontend nhận ảnh dưới dạng
**data-URI** qua `GetIconData(file)` nên không cần máy chủ web riêng và vẫn hiện được khi
mất mạng. Tên file luôn phải là **tên đơn giản** — `internal/icon.ValidName` chặn `..` và dấu
phân tách đường dẫn.

Sinh lại icon ứng dụng cho Windows:

```bash
go run ./cmd/genicon     # -> build/appicon.png + build/windows/icon.ico
```

## Nguyên tắc kiến trúc

**Go giữ bí mật, frontend chỉ hiển thị.** Khi két khóa, `GetState()` trả về mask thay vì giá
trị thật và `RevealSecret` / `RevealCustomPass` từ chối. Sửa logic bảo mật ở một chỗ ở Go,
đừng vá từng nút copy ở JS.

Mọi dữ liệu người dùng chèn vào HTML phải qua `esc()` (text node) hoặc `escJs()` (chuỗi
trong `onclick`).

## Tích hợp tuỳ chọn

Dịch vụ tạo ảnh **mặc định tắt**. Bật bằng cách copy `integrations.example.json` sang
`%AppData%\SubMailPersonal\integrations.json` rồi điền `baseUrl` / `apiKey` của bạn và
đặt `"enabled": true`. Chi tiết: [`../README.md`](../README.md#tích-hợp-tuỳ-chọn-dịch-vụ-tạo-ảnh).

## Giao diện

Frontend dùng **CSS gốc** trong `frontend/src/style.css` (không Tailwind CDN), bám theo hệ
TailAdmin v2: sidebar cố định + topbar có ô tìm kiếm `Ctrl K`, font Outfit bundle tại
`frontend/src/assets/fonts/` nên chạy hoàn toàn offline.

Ba trang: **Két Gmail** (mật khẩu + mã 2FA sống theo từng hộp thư), **Tổng quan Subs**
(chi phí, gia hạn, cảnh báo) và **Hệ thống** (dữ liệu, sao lưu, bảo mật).

Modal **Thêm Gói Đăng Ký** có lưới **chọn nhanh** ~30 dịch vụ phổ biến lấy từ
`GetIconPresets()`. Chọn một mẫu là tự điền tên, gói, danh mục, giá và phương thức thanh toán.
Danh sách mẫu và prompt sinh ảnh nằm ở `internal/icon/presets.go`.
