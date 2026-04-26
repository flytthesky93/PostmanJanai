# Release checklist

Tick-được từng mục. Một bản Windows x64 chỉ được gọi là "internal release" khi **tất cả** mục ngoài "Tuỳ chọn" đều tick. Ghi rõ version + commit hash + build date ở đầu bảng trước khi tick.

- **Version:** (ví dụ `v0.5.0-internal.1`)
- **Commit:** `git rev-parse --short HEAD` → `__________`
- **Build date (UTC):** `__________`
- **Build by:** `__________`

## 1. Code health

- [ ] `git status` sạch (không có file thay đổi / untracked).
- [ ] Branch đang build là branch chính thức (mặc định `main`).
- [ ] `go vet ./internal/...` không có cảnh báo.
- [ ] `go test ./internal/... -count=1 -timeout 120s` → **PASS** trên máy dev (Windows, CGO tắt).
- [ ] CI workflow xanh (chứa cả `go test ./internal/... -race`) trên commit tương ứng.
- [ ] `npm run build` trong `frontend/` không có lỗi/warning về missing dependency.
- [ ] CI xanh trên commit tương ứng (`.github/workflows/ci.yml`).
- [ ] Không có code chỉnh tay vào `ent/*` (trừ `ent/schema/`, `ent/generate.go`) hoặc `frontend/wailsjs/**`.

## 2. Schema / migration

- [ ] `constant.DBSchemaUserVersion` khớp với số step migration đã implement trong `dbmanage.migrateOneStep`.
- [ ] Nếu có bump version: đã thêm branch mới trong `migrateOneStep` + kèm test ở `internal/dbmanage/data_migrate_test.go`.
- [ ] Thử mở app với 1 file DB cũ (từ bản release trước) → app khởi động thành công, backup được tạo dưới `<appDir>/backups/`.
- [ ] Thử mở app với `appDir` trống (first-run) → DB mới được tạo, `PRAGMA user_version` = giá trị target.

## 3. Build Wails (Windows x64 — platform chính thức v1)

- [ ] `make build-win-safe` chạy xong không lỗi.
- [ ] Binary `build/bin/PostmanJanai.exe` tồn tại, mở được, cửa sổ hiện đúng.
- [ ] Kích thước binary ở mức hợp lý (không phình bất thường so với bản trước).
- [ ] Trên máy sạch (không có Go / Node), app vẫn khởi động (chỉ cần WebView2 runtime của Windows).
- [ ] NSIS installer build được (nếu có thay đổi): `build/windows/installer/` chạy pass.

## 4. Smoke manual (Windows x64)

Chạy bản binary vừa build, thực hiện `manual-test-plan.md` mục **Sanity path (5–10 phút)**. Tất cả step tick "Pass":

- [ ] First-run tạo DB mới; tạo 1 root folder.
- [ ] Tạo env với `{{base_url}}` → active → send request dùng placeholder → nhận response 200.
- [ ] Save request → reload app → request vẫn còn.
- [ ] Import 1 file Postman mẫu; export root folder → import lại file export → tree khớp.
- [ ] History: click 1 request cũ → payload + response khôi phục đúng.
- [ ] Không có panic/log ERROR lạ trong `<appDir>/logs/app.log`.

Nếu release này có thay đổi **Phase 6 (networking/security)** (proxy/CA/TLS/insecure/secret env) → thêm tick nhanh:

- [ ] Tab **Settings** mở được; **Test proxy** không làm crash app.
- [ ] (Tuỳ chọn) proxy/CA theo môi trường có thể test được — xem `manual-test-plan.md` §**I. Networking & security (Phase 6)**.

Nếu release này có thay đổi **Phase 7 (UX polish/productivity)** → thêm tick nhanh:

- [ ] Đóng tab cuối → Dashboard hiện, reload vẫn không tự tạo tab.
- [ ] `Ctrl+K` mở palette; tìm request/folder/env hoạt động.
- [ ] Duplicate request/folder và Copy as cURL hoạt động — xem `manual-test-plan.md` §**J. UX polish & productivity (Phase 7)**.
- [ ] `npm run build` không còn warning chunk > 500 kB sau Vite code splitting.

## 5. Artifacts & release notes

- [ ] Binary + installer (nếu có) upload lên nơi lưu trữ nội bộ (Drive / S3 / Release draft).
- [ ] Release note liệt kê: feature mới, fix, known limitations (nếu có — tham chiếu backlog trong `roadmap.md`).
- [ ] `roadmap.md` + `data-model-and-delivery-status.md` đã được cập nhật cho bản release này.

## 6. Tuỳ chọn (không chặn release)

- [ ] Thử chạy trên Linux (`make build-linux`) / macOS (`make build-mac-universal`) — nếu chạy được, ghi chú vào release note dạng "best-effort, chưa ký".
- [ ] Code signing Windows (nếu đã có cert).
- [ ] Notarize macOS (nếu build macOS).
- [ ] Bench kích thước DB / throughput sau 500 saved request (ghi vào release note).

---

**Rollback plan:** bản release nội bộ được zip + giữ ít nhất 2 bản gần nhất. Nếu phát hiện bug nghiêm trọng sau khi phát hành, người dùng có thể quay lại bản trước (DB có backward-compat thông qua `user_version` migration forward-only; downgrade phải restore DB backup từ `<appDir>/backups/`).
