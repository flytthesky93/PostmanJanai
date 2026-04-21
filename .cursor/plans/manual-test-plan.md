# Manual test plan — PostmanJanai

Lộ trình test tay cho 1 bản **Windows x64** build bằng `make build-win-safe`. Dùng song song với [release-checklist.md](release-checklist.md).

Quy ước:

- **Thiết bị**: Windows 10/11 x64 sạch (không có Go, Node, Postman cũ chạy kèm).
- **Preconditions**: close app hoàn toàn trước mỗi nhóm test. `appDir` của PostmanJanai thường ở `%APPDATA%\PostmanJanai` — check log tại `<appDir>/logs/app.log`.
- **Kết quả**: Pass / Fail / Blocked + link / screenshot / log trích nếu Fail.
- **Tester** điền tên + ngày ở mỗi phần.

---

## Sanity path (5–10 phút) — bắt buộc pass cho mọi release

> Dùng để "smoke" nhanh khi tick release checklist §4. Nếu bất kỳ bước nào fail → không release.

1. **First-run tạo DB**
   - Xoá `<appDir>/PostmanJanai.db` (nếu có) → mở app.
   - **Expect**: app khởi động, cửa sổ maximize, sidebar rỗng. Log có dòng `Application started ...`.
2. **Tạo root folder "Smoke"**
   - Sidebar → nút "Thêm folder" → tên `Smoke` → lưu.
   - **Expect**: folder hiện lên, được chọn sẵn.
3. **Tạo environment "Local"**
   - Chuyển tab Environments → `+ Environment` → tên `Local`.
   - Thêm 2 biến: `base_url=https://httpbin.org`, `token=demo`. Enable cả hai.
   - Bấm "Set active".
4. **Gửi request dùng `{{base_url}}`**
   - Quay lại sidebar, chọn folder `Smoke` → `+ Request`.
   - Method `GET`, URL `{{base_url}}/get?x=1`, bấm **Send**.
   - **Expect**: status 200; response body có field `args.x = "1"`; `{{base_url}}` được hiển thị resolved trong History.
5. **Save request**
   - Bấm "Save" → đặt tên `demo-get` → lưu.
   - **Expect**: request nằm trong folder `Smoke` trên sidebar.
6. **Reload persistence**
   - Đóng app hoàn toàn → mở lại.
   - **Expect**: folder `Smoke`, env `Local` (active), request `demo-get` vẫn còn.
7. **Import → Export → Import**
   - Import 1 file Postman v2.1 mẫu (giữ trong `testdata/` hoặc file nhỏ bất kỳ có 1 folder + 2 request).
   - Bấm "Export" trên root vừa import → lưu file JSON.
   - Import lại chính file vừa export.
   - **Expect**: xuất hiện root mới `<tên collection> (2)`, tree bên trong giống hệt (folder names + request names + method + URL).
8. **History**
   - Chuyển tab History, click vào row `demo-get` ở bước 4.
   - **Expect**: headers / body / response khôi phục đầy đủ.
9. **Không có lỗi lạ**
   - Mở `<appDir>/logs/app.log` (tail 200 dòng cuối).
   - **Expect**: không có `ERROR`/`panic`/`fatal` trong khi chạy sanity path.

**Sanity path passed bởi:** ________________  **Ngày:** __________

---

## Full regression (40–60 phút) — chạy khi bump phase hoặc chạm schema/migration

Chia theo domain. Mỗi mục là một hàng tick "Pass/Fail/N/A" + ghi chú.

### A. Folder (sidebar tree)

1. Tạo root folder mới; rename; xoá.
2. Tạo folder con 2 cấp (nested); rename; xoá (assert cả subtree bị xoá).
3. Trùng tên root → hiện message `FOL_301`.
4. Trùng tên dưới cùng parent → hiện message `FOL_302`.
5. **Drag-and-drop reorder**: kéo một folder con lên/xuống dưới cùng parent → state giữ sau reload.
6. **DnD move**: kéo folder sang parent khác; kéo ra root.
7. **DnD move chặn cycle**: kéo root vào chính con của nó → hiện message cảnh báo, không thực thi.
8. Expand/collapse folder → state giữ sau reload (localStorage).

### B. Saved request

1. Create với mỗi body mode: `none`, `raw`, `xml`, `form_urlencoded`, `multipart` (có file + text part).
2. Auth: none → bearer (với `{{token}}`) → basic → apikey (header + query). Save & reload → giá trị khớp.
3. Duplicate tên trong cùng folder → block với message `REQ_502`.
4. Move request sang folder khác qua DnD; kiểm tra đích không có trùng tên (nếu có, phải block).
5. Rename request → sidebar + tab cập nhật.
6. Multi-tab: mở 3 request khác nhau cùng lúc → chuyển tab không mất dirty state.
7. Dirty marker: sửa 1 request nhưng không save → chuyển app đi và quay lại → dirty vẫn còn (localStorage).
8. Đóng tab dirty → xác nhận prompt.

### C. Environment

1. CRUD env.
2. Trùng tên env → `ENV_602`.
3. Duplicate variable key (case-insensitive) → `ENV_603`.
4. Toggle enabled của 1 biến → biến disabled KHÔNG được áp dụng khi Send.
5. Set active qua lại giữa 2 env → sidebar hiển thị đúng biểu tượng active, chỉ một env được active tại mọi thời điểm.
6. Clear active → `{{var}}` trong URL hiển thị nguyên văn khi Send (network fail / URL sai).

### D. HTTP execution

1. Gửi GET http / https (qua `httpbin.org/get`).
2. Gửi POST raw JSON; xác minh server nhận đúng body.
3. Gửi multipart với 1 file nhỏ (< 1MB) + 1 text part.
4. Timeout: URL không tồn tại / port đóng → result ghi nhận `ErrorMessage`, UI hiển thị friendly error.
5. Response body lớn (> cap): bật `httpbin.org/bytes/20000000` → body bị truncate và UI hiển thị `[… response body truncated …]`.
6. Auth merge: bearer, basic (xem Authorization header), apikey in=header, apikey in=query → header/query đến đúng server.

### E. History

1. Mỗi lần Send → 1 row history mới, order mới nhất trên cùng.
2. Click vào row → khôi phục cả request body + response body.
3. Filter theo root folder: chọn 1 folder ở sidebar → history chỉ hiển thị request của folder đó.
4. Delete 1 row history → row biến mất, row khác không ảnh hưởng.
5. Xoá folder chứa request đã có history → history giữ nguyên, badge "origin gone" / FK cleared (không link được về folder/request cũ).

### F. Import / Export

1. Import Postman v2.1 collection (file thật, có nested folders + auth + body raw).
2. Import Postman v2.0 (legacy).
3. Import OpenAPI 3.x JSON.
4. Import OpenAPI 3.x YAML.
5. Import Insomnia v4.
6. Import file rỗng → `IMP_703`.
7. Import file > 25MB → `IMP_702`.
8. Import JSON không phải format hỗ trợ → `IMP_704`.
9. Import với option `Create environment` + `Activate` → env mới xuất hiện và active ngay.
10. Export Postman v2.1 từ root folder → file hợp lệ (mở được trong Postman desktop).
11. Re-import file vừa export → tree khớp (tên root mới thêm `(2)`).

### G. Search

1. Search chuỗi có trong tên folder → click hit → sidebar expand tới folder đó.
2. Search chuỗi có trong tên/URL request → click hit → request mở trong tab.
3. Kết quả > limit → banner "truncated" hiện ra.
4. Query trống / chỉ whitespace → không gọi backend, UI giữ trạng thái cũ.

### H. Snippet

1. Mở 1 request với bearer auth + body JSON → generate snippet cho từng target: curl / fetch / axios / httpie.
2. Snippet có substitute `{{var}}` → xem trong curl phải là giá trị thật (không phải `{{var}}`).
3. Copy-to-clipboard hoạt động.

### I. Migration & backup

1. Từ DB v4 (lấy từ backup bản release trước) → mở app → `<appDir>/backups/` có file backup mới, app chạy bình thường, folders có `sort_order` backfill đúng (alphabetical trong từng parent).
2. Từ DB đang ở version target → mở app → không tạo backup mới (tránh spam).
3. DB bị corrupt (cố tình ghi byte rác): app không panic — phải show error message thân thiện hoặc ít nhất log có stack trace rõ ràng (known limitation: hiện tại có thể fatal; ghi chú vào release note nếu vẫn xảy ra).

### J. Crash / stability

1. Chạy app nhiều giờ (> 2h) với 1 vài request định kỳ → memory không leak quá mức (Task Manager: < 500MB).
2. Đóng app đột ngột (Task Manager kill) → lần mở sau vẫn bình thường.

---

## Lần cuối điền trước khi phát hành

- **Tester**: ________________
- **Bắt đầu (UTC)**: __________ **Kết thúc (UTC)**: __________
- **Tóm tắt**: Pass __/Fail __/Blocked __
- **Bug cần fix trước release**: (link issue)
- **Known issues chấp nhận cho bản này**: (link backlog trong roadmap.md)
