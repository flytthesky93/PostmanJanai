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

10. **(Tuỳ chọn — nhanh, nếu release có thay đổi Phase 6)** Settings smoke
   - Mở tab **Settings** → Proxy mode `none` → **Test proxy** tới `https://example.com` → có kết quả (OK hoặc lỗi có message), app không crash.
   - Environments → tạo biến `x` kind **secret** → Save → reload app → value vẫn là secret (masked), Send dùng `{{x}}` vẫn resolve đúng.

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

### I. Networking & security (Phase 6)

> Cần môi trường có proxy/SSL inspection thật **hoặc** tự dựng proxy/HTTPS test nội bộ. Nếu không có điều kiện → tick **N/A** + lý do.

1. **Proxy — system**
   - Set Windows env `HTTP_PROXY`/`HTTPS_PROXY` trỏ tới 1 proxy hợp lệ → trong app Settings chọn **system** → Send `GET https://httpbin.org/get` → 200.
2. **Proxy — manual + auth (nếu proxy yêu cầu user/pass)**
   - Settings → manual: điền URL proxy + username + password → **Test proxy** tới `https://example.com` → OK.
   - Send request ra ngoài qua proxy → 200 (hoặc lỗi có message rõ, không crash).
3. **NO_PROXY**
   - Điền `NO_PROXY` khớp host đích (ví dụ `httpbin.org`) → request tới host đó không đi qua proxy (xác minh bằng log proxy hoặc tcpdump nếu có).
4. **Custom CA**
   - Import PEM (hoặc pick file) của CA đang sign cert server nội bộ → bật enabled → gọi HTTPS tới host đó → không còn lỗi `x509: certificate signed by unknown authority`.
5. **Insecure skip verify (per-request)**
   - Bật toggle trên request → Send tới HTTPS self-signed (môi trường test) → thành công; tab + history row có badge **insec**.
6. **Secret env var + history redact**
   - Tạo biến `token` kind **secret**, dùng `Authorization: Bearer {{token}}` → Send → mở History chi tiết: token **không** xuất hiện dạng plaintext (chỉ `***` hoặc không hiện).

### J. UX polish & productivity (Phase 7)

1. **Dashboard + close last tab**
   - Mở vài tab request → đóng hết tab.
   - **Expect**: không tự tạo tab rỗng; Dashboard hiện Recent / Quick actions / Stats; reload app vẫn ở trạng thái Dashboard.
2. **Dashboard quick actions**
   - Từ Dashboard bấm New Request, New folder, Import collection, Import cURL, New environment.
   - **Expect**: mỗi action mở đúng flow hiện có, không làm app crash.
3. **Command palette**
   - `Ctrl+K` → gõ một phần tên request → Enter.
   - **Expect**: request mở trong tab. Gõ tên folder → Enter → sidebar reveal folder. Gõ tên environment → Enter → mở editor env.
4. **Keyboard shortcuts**
   - `Ctrl+Enter` gửi request active; `Ctrl+S` save/update; `Ctrl+T` tạo tab; `Ctrl+W` đóng tab; `Esc` đóng palette/settings.
   - **Expect**: shortcut không phá typing trong input; `Ctrl+W` không đóng cửa sổ app.
5. **Help modal**
   - Bấm nút **?** trên thanh top bar.
   - **Expect**: modal hướng dẫn hiện shortcuts + productivity tips; đóng được bằng Close / click backdrop / `Esc`.
6. **Variable preview**
   - Active env có `base_url`, `id`, và `token` kind secret. URL `{{base_url}}/users/{{id}}`, body raw chứa `{{token}}`.
   - **Expect**: preview resolved đúng; secret hiển thị `***`; biến thiếu báo unresolved.
7. **Duplicate folder/request**
   - Duplicate 1 request có headers/query/body/auth/TLS flag.
   - Duplicate 1 folder có nested folder + request.
   - **Expect**: bản copy nằm cùng parent, tên dạng `(copy)` / `(copy 2)`, payload và tree được giữ, bản gốc không đổi.
8. **Copy as cURL**
   - Với request active có env/auth/body, bấm **Copy cURL** rồi paste vào terminal.
   - **Expect**: cURL chạy tương đương request trong app; console báo copy thành công.

### K. Collection Runner & Chaining (Phase 8)

> Tiền đề: env active có `base_url` trỏ tới 1 server thật (httpbin.org) hoặc mock cục bộ; tạo folder `Phase8` với 2 saved request: `01 Login` (POST `{{base_url}}/post`, body trả `{"token":"<x>"}` qua field) và `02 Me` (GET `{{base_url}}/headers` dùng bearer `{{TOKEN}}`).

1. **Capture rule (single Send)**
   - Trên `01 Login` → tab **Captures** → thêm: name `token`, source `json_body`, expression `$.json.token` (hoặc field tương ứng), target scope `environment`, target var `TOKEN`, enabled. Save.
   - Send → tab **Tests** ở ResponsePanel hiện capture với value đúng; mở Environments → biến `TOKEN` đã được ghi vào env active.
2. **Assertion rule (single Send)**
   - Trên `01 Login` → tab **Tests** (RequestPanel) → assertion `status eq 200`. Save → Send.
   - **Expect**: ResponsePanel tab **Tests** hiển thị `1/1` PASS, summary pill xanh.
   - Đổi assertion thành `status eq 999` → Send → hiển thị `0/1` FAIL với message rõ.
3. **Chaining qua Runner**
   - Trên `02 Me` → header `Authorization: Bearer {{TOKEN}}` (hoặc Bearer auth = `{{TOKEN}}`).
   - Header App → bấm **Runner** (hoặc context menu folder `Phase8` → **Run folder…**) → chọn env active → Run.
   - **Expect**: progress stream hiển thị từng request done theo thứ tự; tổng `2 passed / 0 failed`; request `02 Me` thấy header `Authorization: Bearer <giá trị token vừa capture>`.
4. **Stop on fail + Cancel**
   - Bật toggle **Stop on fail** + thêm 1 request lỗi (URL sai) ở giữa danh sách → Run → dừng đúng request lỗi, status run = `failed`.
   - Run lại folder lớn (≥ 5 request) → bấm **Cancel** giữa chừng → run dừng, status = `cancelled`, recent runs ghi nhận đúng.
5. **Recent runs + Detail**
   - Mở section **Recent runs** trong Runner modal → thấy ≥ 3 entry mới nhất (tên folder, env, totals, duration). Click 1 entry → load lại bảng kết quả.
   - Xoá 1 entry → biến mất khỏi list.
6. **Export report**
   - Sau 1 lần run xong → bấm **Export JSON** → Save dialog mở, save → file `.json` hợp lệ (mở thấy `passed_count`, `requests[]`).
   - Bấm **Export Markdown** → file `.md` đọc được, có bảng totals + section per-request, escape pipe trong URL không phá table.
7. **Migration v6 → v7**
   - Mở app với DB v6 (backup từ build trước Phase 8) → app khởi động không lỗi, `<appDir>/backups/` có file backup, các bảng cũ giữ nguyên dữ liệu, các bảng mới (`request_captures`/`request_assertions`/`runner_runs`/`runner_run_requests`) tồn tại.
8. **Cascade delete**
   - Tạo capture + assertion cho 1 request → xoá request đó → mở DB browser (hoặc tạo lại request cùng id không khả thi vì UUID mới) → đảm bảo các row capture/assertion đã được xoá, không còn orphan.
   - Xoá nguyên folder chứa request có rules → tương tự, không còn orphan.

### L. Migration & backup

1. Từ DB v4 (lấy từ backup bản release trước) → mở app → `<appDir>/backups/` có file backup mới, app chạy bình thường, folders có `sort_order` backfill đúng (alphabetical trong từng parent).
2. Từ DB đang ở version target → mở app → không tạo backup mới (tránh spam).
3. DB bị corrupt (cố tình ghi byte rác): app không panic — phải show error message thân thiện hoặc ít nhất log có stack trace rõ ràng (known limitation: hiện tại có thể fatal; ghi chú vào release note nếu vẫn xảy ra).

### M. Crash / stability

1. Chạy app nhiều giờ (> 2h) với 1 vài request định kỳ → memory không leak quá mức (Task Manager: < 500MB).
2. Đóng app đột ngột (Task Manager kill) → lần mở sau vẫn bình thường.

---

## Lần cuối điền trước khi phát hành

- **Tester**: ________________
- **Bắt đầu (UTC)**: __________ **Kết thúc (UTC)**: __________
- **Tóm tắt**: Pass __/Fail __/Blocked __
- **Bug cần fix trước release**: (link issue)
- **Known issues chấp nhận cho bản này**: (link backlog trong roadmap.md)
