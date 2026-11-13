# Đóng góp

Cảm ơn bạn đã muốn cải thiện SubMail Personal!

## Trước khi bắt đầu

- Mở issue trước khi làm thay đổi lớn, để tránh hai người cùng làm một việc.
- Với thay đổi nhỏ (sửa chữa chính tả, thêm một mục preset), cứ tạo pull request thẳng.

## Chạy ứng dụng ở chế độ phát triển

```bash
cd desktop
wails dev
```

## Kiểm thử trước khi gửi pull request

```bash
cd desktop
gofmt -l .        # không được có output
go vet ./...
go test ./...

cd frontend
npm test
```

Tất cả lệnh trên phải xanh. Không thêm framework test mới (testify, jest, vitest…):
repo dùng `go test` và `node --test` thuần.

## Quy ước trong mã

| Rủi ro | Quy ước |
|---|---|
| Ngôn ngữ | Chuỗi hiển thị trên UI: **tiếng Việt**. Comment trong mã (Go và JS): **tiếng Anh**. |
| Bảo mật | Bí mật chỉ nằm ở tầng Go. Frontend **không bao giờ** nhận `password` / `secret2fa` / `backupCodes` / `customPass` khi két khóa. |
| XSS | Mọi text node dùng `esc(value)`; mọi chuỗi trong `onclick` dùng `escJs(value)`. Không bỏ qua bước này. |
| Luồng ghi | `sửa state → persist() → trả Snapshot → frontend applySnapshot() + refreshAll()`. Thiếu `persist()` là mất dữ liệu khi đóng app. |
| Tập giá trị | `status`, `authMode`, `category`, `color` là tập đóng. Thêm giá trị mới thì phải cập nhật **cả ba** map màu ở `frontend/src/main.js`, nếu không sẽ in ra `undefined` trong HTML. |
| Tiền tệ | Luôn hiện **cả USD lẫn VNĐ** qua `formatVnd(usd)`. |
| Màu | Mọi màu đi qua CSS custom property. Thêm biến mới thì khai báo **cả hai** block `:root` và `html.dark` trong `frontend/src/style.css`. |
| Font | Font Outfit được bundle. Phải giữ subset `latin-ext` — dấu tiếng Việt nằm ngoài subset `latin`. |

## Bất biến kiến trúc cần giữ

1. **Go giữ bí mật, frontend chỉ hiển thị.** Sửa logic bảo mật ở *một chỗ* ở Go, đừng vá
   từng nút copy ở JS.
2. **Ngày gia hạn luôn tính lại ở Go** (`renewalText` / `renewalUrgency` trong `app.go`), không
   lưu `nextText` / `urgency` vào file.
3. **`paused` bị loại khỏi mọi phép tính chi phí và cảnh báo.** Mọi nơi lọc
   `status !== 'paused'`.
4. **Xoá một Gmail phải xoá cả các subscription** tham chiếu tới nó.

## Thêm tích hợp mới

Tích hợp là **tuỳ chọn** và **mặc định tắt**:

- Thêm loại mới trong `desktop/internal/integrations/`.
- Đọc cấu hình từ file cục bộ; **không** gửi `apiKey` lên frontend.
- `App.GetIntegrations()` chỉ trả `id`, `name`, `type`, `enabled`.
- Nút trên UI chỉ hiện khi tích hợp tương ứng được bật (xem `loadIntegrations` trong
  `desktop/frontend/src/main.js`).

## Commit message

Viết bằng tiếng Việt hoặc tiếng Anh, mô tả **việc làm** chứ không mô tả cách làm:

```text
feat(icons): thêm nút tạo icon cho gói đăng ký
fix(vault): chặn copy mã dự phòng khi két đang khóa
docs(readme): hướng dẫn cấu hình dịch vụ tạo ảnh
```

## Báo lỗi

Mở issue kèm: bước tái hiện, phiên bản, hệ điều hành, ảnh chụp màn hình nếu có.
