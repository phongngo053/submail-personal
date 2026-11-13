# Chính sách bảo mật

## Phạm vi

SubMail Personal là ứng dụng desktop **chạy hoàn toàn trên máy của bạn**. Nó không có máy
chủ, không có tài khoản người dùng, không gửi dữ liệu đi đâu.

## Bảo mật nằm ở đâu

| Nội dung | Vị trí | Dạng lưu |
|---|---|---|
| Mật khẩu, secret 2FA, mã dự phòng, App Password | `%AppData%\SubMailPersonal\data.json` | **Rõ** (không mã hoá) |
| File sao lưu do bạn xuất | Bất kỳ đâu bạn chọn | **Rõ** (không mã hoá) |
| Ảnh icon đã tạo | `%AppData%\SubMailPersonal\icons\` | PNG thường |
| `apiKey` của dịch vụ tạo ảnh | `%AppData%\SubMailPersonal\integrations.json` | **Rõ** |

## Những điều app **không** làm

- App **không mã hoá at-rest**. File `data.json` đọc được bằng bất kỳ trình soạn text nào.
- Nút **"Khóa Két"** chỉ che hiển thị ở tầng giao diện và chặn các nút sao chép. Nó **không
  phải** một tính năng mã hoá và không bảo vệ dữ liệu trên đĩa.
- App **không** gửi dữ liệu của bạn lên bất kỳ đâu.
- App **không** tự động bật dịch vụ tạo ảnh. Chỉ khi bạn tự cấu hình
  `integrations.json` thì app mới gọi tới `baseUrl` mà bạn chỉ định.

## Khuyến nghị

1. **Bật mã hoá ổ đĩa** (BitLocker trên Windows, FileVault trên macOS, LUKS trên Linux).
2. **Không đưa vào Git** các file `data.json`, file sao lưu hay `integrations.json`. Repo
   này đã ignore sẵn, nhưng hãy kiểm tra lại trước khi commit.
3. **Không chia sẻ file sao lưu** cho người khác — chúng chứa bí mật ở dạng rõ.
4. **Xóa dữ liệu** khi không dùng nữa: bấm **Xóa toàn bộ dữ liệu** ở trang *Hệ thống*, rồi
   xoá thư mục `%AppData%\SubMailPersonal`.
5. Dùng ứng dụng này cho **tài khoản phụ**. Mật khẩu chính của bạn nên nằm trong trình
   quản lý mật khẩu chuyên dụng.

## Báo lỗi bảo mật

Nếu bạn phát hiện lỗi bảo mật, **đừng** mở issue công khai. Hãy báo riêng qua:

- GitHub Security Advisories của repo (tab **Security** → **Report a vulnerability**), hoặc
- email: `ngotienphong5141@gmail.com`

Vui lòng gửi kèm: bước tái hiện, phiên bản, hệ điều hành. Cho bạn **30 ngày** để xử lý
trước khi công bố.

## Phạm vi

Các hành vi **không** được coi là lỗi bảo mật:

- Dữ liệu lưu dạng rõ trên đĩa (thiết kế hiện tại, đã ghi rõ ở trên).
- Nút "Khóa Két" không mã hoá (cũng đã ghi rõ).
- Người dùng tự cấu hình `integrations.json` trỏ tới một dịch vụ tuỳ ý.
