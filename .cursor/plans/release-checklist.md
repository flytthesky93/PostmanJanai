# Release checklist

> **Plan v1 đã đóng backlog.** Checklist này là **cổng chất lượng theo từng bản build**, không phải danh sách backlog tính năng. Ý tưởng sau Phase 9 ghi trong **phase mới** tại `roadmap.md`.

Tick-được từng mục — ghi **version**, **commit hash**, **build date (UTC)** ở đầu bảng trước khi tick đủ. Một bản Windows x64 chỉ gọi **internal release** khi **tất cả** mục ngoài "Tuỳ chọn" đều tick.

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

Nếu release này có thay đổi **Phase 8 (Collection Runner & Chaining)** → thêm tick nhanh:

- [ ] DB mở từ bản v6 cũ → migrate chain v6→v7→v8 chạy thành công, các bảng mới `request_captures` / `request_assertions` / `runner_runs` / `runner_run_requests` tồn tại; cột v8 (`request_headers_json` / `response_headers_json` / `request_body` / `response_body` / `body_truncated`) đã có; bảng cũ không bị đụng.
- [ ] Tạo capture `$.token` → environment scope → request kế tiếp dùng `{{token}}` resolve đúng giá trị mới (chained).
- [ ] Assertion `status eq 200` PASS hiện trong tab **Tests** của ResponsePanel; assertion FAIL hiển thị message rõ.
- [ ] Mở **Runner** trên header (hoặc context menu folder → "Run folder…") → run xong total/passed/failed khớp; recent runs hiện đúng.
- [ ] Stop-on-fail dừng đúng request lỗi; cancel run đang chạy hoạt động (status `cancelled`); cancel trong lúc đang **delay** cũng dừng kịp thời.
- [ ] **Export JSON** + **Export Markdown** từ Runner modal mở Save dialog, lưu file đọc được — xem `manual-test-plan.md` §**K. Collection Runner & Chaining (Phase 8)**.
- [ ] (Phase 8.1) Click 1 row trong run report → modal chi tiết hiện raw resolved headers + body request/response (không còn `{{var}}`); response body lớn có suffix `[… response body truncated …]` khớp History detail.
- [ ] (Phase 8.1) Iterations = 3, DelayMs = 200, Timeout/req = 1000 → run xong tổng row = 3 × số request, recent runs ghi nhận đúng tổng; nhập Iterations = 999 → backend tự clamp về 50.

Nếu release này có thay đổi **Phase 9 (Scripting)** — **Phase 9.0 closed in repo 2026-04-30**; các mục dưới đây vẫn là **gate thủ công** cho bản chứa Scripting (coi là regression smoke):

- [ ] DB mở từ bản v8 cũ → migrate v8→v9 chạy thành công, `requests.pre_request_script` + `requests.post_response_script` đã có (default `''`); rerun lần 2 không tạo backup mới, không lỗi duplicate column.
- [ ] Tab **Pre-request** + **Post-response** + **Assertions** trong RequestPanel mở được; nhập script `pmj.environment.set('token', …)` (hoặc alias `pm`) ở Post-response → Send → biến `token` cập nhật trong env active.
- [ ] Script vô hạn (`while(true){}`) bị kill sau timeout, app không treo, console hiển thị error rõ.
- [ ] Script `require('fs')` hoặc gọi I/O bị block / throw `ReferenceError`, không gây crash.
- [ ] Runner chạy folder có script: pre-request fail → request bị skip, mark error; post-response fail → mark fail; `pm.test()` được rollup vào tổng pass/fail.
- [ ] Import 1 collection Postman có `event[]` (`prerequest` / `test`) → script chạy được trong Runner ở mức `pm.*` subset; Export lại → script text y nguyên — xem `manual-test-plan.md` §**L. Scripting (Phase 9)**.

## 5. Artifacts & release notes

- [ ] Binary + installer (nếu có) upload lên nơi lưu trữ nội bộ (Drive / S3 / Release draft).
- [ ] Release note liệt kê: feature mới, fix, known limitations có **issue hoặc phase** rõ ràng (nếu có).
- [ ] `roadmap.md` + `data-model-and-delivery-status.md` phản ánh phiên bản / phase có thay đổi trong bản release này.

## 6. Tuỳ chọn (không chặn release)

- [ ] Thử chạy trên Linux (`make build-linux`) / macOS (`make build-mac-universal`) — nếu chạy được, ghi chú vào release note dạng "best-effort, chưa ký".
- [ ] Code signing Windows (nếu đã có cert).
- [ ] Notarize macOS (nếu build macOS).
- [ ] Bench kích thước DB / throughput sau 500 saved request (ghi vào release note).

---

**Rollback plan:** bản release nội bộ được zip + giữ ít nhất 2 bản gần nhất. Nếu phát hiện bug nghiêm trọng sau khi phát hành, người dùng có thể quay lại bản trước (DB có backward-compat thông qua `user_version` migration forward-only; downgrade phải restore DB backup từ `<appDir>/backups/`).
