# PostmanJanai — Roadmap

**Spec DB + checklist triển khai kỹ thuật:** [data-model-and-delivery-status.md](data-model-and-delivery-status.md)

---

## Product Goal

Build a desktop API client (Postman-like) focused on:

- Reliable HTTP request execution.
- **Folder** (nested) + **Request** management — folders thay cho workspace + collection; cây lồng nhau không giới hạn độ sâu (trong phạm vi UI hiện tại).
- Request history and debugging.
- Auth and environment variables.
- Smooth developer workflow for daily use.

## Phase completion log

| Phase | Status     | Notes (closing snapshot) |
|-------|------------|---------------------------|
| **0** | **Done**   | Closed **2026-04**: foundation stable for local-only app; see Phase 0 section below. |
| **1** | **Done**   | Closed **2026-04-17**: runner thật (`HTTPExecutor` + `HTTPHandler`), editor request/response/console, history persist + list; gắn **root folder context** khi gửi (`root_folder_id`); import request từ cURL. Chi tiết: [data-model-and-delivery-status.md](data-model-and-delivery-status.md). |
| **2** | **Done**   | Closed **2026-04**: CRUD **folder** (cây lồng) + **saved request** (`folder_id`), UI sidebar **Folders** + cây đệ quy (`FolderCatalog` / `FolderTreeNode`), Wails `FolderHandler` + `SavedRequestHandler`, bump **DB v3** (workspace/collection → folder). |
| **3** | **Done**   | Closed **2026-04-19**: `EnvironmentHandler` + CRUD env/biến + **một env active**; substitute `{{var}}` trước `HTTPExecutor` (URL/body/headers/query/form/multipart/auth); auth **none / bearer / basic / apikey** (header hoặc query), lưu `auth_json` trên request; history lưu **payload đã resolve**; UI history chi tiết (modal snapshot); editor `{{var}}` (chip, popover, caret “atomic” trên CodeMirror + `EnvVarMirrorField`). Chi tiết: [data-model-and-delivery-status.md](data-model-and-delivery-status.md). |
| **4** | **Done** (2026-04-20) | **Productivity scope đã đủ để đóng:** #1 Import; #2 Multi-tab; #3 Search/filter; Flow B — export Postman v2.1, snippets, cây folder (expand persist, rename, DnD move + **reorder** + vùng **Same level / Inside** + root drop), **DB v5** `folders.sort_order`. **Polish cùng ngày:** click mở/thu **full hàng** + khe reorder; rename folder **chỉ ⋮** (không double-click). **Backlog tùy chọn (không chặn Phase 4):** export project JSON “native”; migrate v2→v3 giữ dữ liệu. |
| **5** | **Done** (2026-04-21) | 4 nhóm đo được đã giao: (1) test bù lấp (dbmanage + repository + usecase + import/export round-trip); (2) smoke E2E tầng Go (`internal/e2e/smoke_test.go`); (3) CI tối thiểu `.github/workflows/ci.yml` (go vet + go test -race + vite build) — **xanh trên GitHub Actions 2026-04-21**; (4) `release-checklist.md` + `manual-test-plan.md`. **Windows-first:** v1 official = Windows x64; macOS/Linux best-effort, unsigned. Backlog không nằm trong Phase 5: export project JSON native; migration v2→v3 giữ dữ liệu; UI E2E (Playwright); bug `{{var}}` bị URL-encode khi export Postman v2.1. |
| **6** | **Done** (2026-04-21) | **Networking & Security:** proxy (`none/system/manual`, URL + username + password ciphertext + `NO_PROXY`), custom CA (`trusted_cas` PEM trong DB), `HTTPTransportFactory` + per-request `insecure_skip_verify` trên `requests`, env var `kind=secret` + AES-GCM `enc:v1:` + redact history/snippets, Wails `SettingsHandler` + tab **Settings** UI + test proxy. **DB v6.** |
| **7** | **Done** (2026-04-26) | **UX Polish & Productivity:** Dashboard khi không còn tab + đóng tab cuối; in-app Help `?`; Command palette Ctrl/Cmd+K; variable interpolation preview có mask secret; duplicate folder/request; Copy as cURL; keyboard shortcuts cơ bản; Vite code splitting để hết warning chunk > 500 kB. Không bump DB. |
| **8** | **Done** (2026-04-26 — closed in two slices, 8.0 + 8.1) | **Collection Runner & Chaining.** *8.0:* schema **DB v7** (`request_captures`, `request_assertions`, `runner_runs`, `runner_run_requests`); capture engine (JSONPath/regex/header/status) + assertion engine (status/header/json-path/duration/size/regex, ops eq/neq/contains/...); HTTPHandler chạy capture+assertion sau Send (saved request); `RunnerUsecase.RunFolder` tuần tự theo folder/sort_order với env active + memory bag, persist `runner_runs` + per-request rows, stream Wails events `runner:started/request/finished`; UI: tab **Captures** + **Tests** trong RequestPanel, summary tests trong ResponsePanel, **Runner** modal (config form, live progress, recent runs, cancel, export JSON/Markdown qua Save dialog), folder context-menu eligibility (chỉ folder chứa request mới enable "Run folder…"). *8.1:* **DB v8** thêm 5 cột vào `runner_run_requests` (`request_headers_json`, `response_headers_json`, `request_body`, `response_body`, `body_truncated`) → lưu raw resolved request/response để xem lại không cần re-run; modal `RunnerRequestDetailModal` hiển thị headers + JSON body có truncate suffix khớp History; runner options đầy đủ promise của roadmap: **Iterations** (≤ `RunnerMaxIterations`=50), **DelayMs** giữa requests (≤ 60s, cancel-aware), **TimeoutPerRequestMs** (override per-Execute, ≤ 5 phút). Tests: capture/assertion/runner_report unit + repo (`request_rule_repository_test`) + 4 unit tests cho options (iterations & delay, clamp, timeout, smoke chaining) + migration tests `TestMigrate_6to7IsAdditive` / `TestMigrate_7to8AddsRunnerRequestSnapshots` / `TestMigrate_6to8Chain` + smoke E2E `phase8_runner_smoke_test` (capture chain `$.token` → env, assertion 2xx, event order, report, raw request/response persisted). |
| **9** | **Done** (2026-04-30) | **Scripting (pre-request & post-response).** goja sandbox + timeouts; globals **`pmj`** trong product (**`pm`** alias trong VM — import Postman cũ). Subset PM API: env/vars/request/response/`pm.test`/`pm.expect`/`pm.sendRequest` (sync). **DB v8 → v9** (`requests.pre_request_script`, `requests.post_response_script`). **HTTPExecuteInput** gửi kèm script từ editor (adhoc / dirty có chạy; ưu tiên payload > DB). **RequestPanel**: tab **Pre-request**, **Post-response**, **Captures**, **Assertions** luôn hiện (layout đồng nhất). **Khôi phục tabs** sau restart: hydrate sau khi **`RequestPanel` async** expose `hydrate()` (không còn lệch tên tab vs nội dung). Runner: pre → HTTP → post → capture/assertion; import/export Postman v2.1 `event[]`. Chi tiết: [data-model-and-delivery-status.md](data-model-and-delivery-status.md) §Phase 9. |

---

### Phase 0 - Stabilize Foundation

Scope:

- Stabilize logging, app-data path, and DB migration behavior.
- Normalize frontend/backend data contracts (naming and payload shape).
- Add consistent UI-side error handling.

Done when:

- **Root folder** CRUD (trước đây “workspace”) stable — *hiện là folder gốc trong mô hình folder.*
- Logs are written reliably (`app.log` and `debug.log`).
- Build and run flow is documented and repeatable.

**Delivered (as of close):**

- **Logging & paths:** `slog` + rotating files (`app.log` / `debug.log`); `POSTMANJANAI_APP_DIR` / `POSTMANJANAI_DB_PATH`; README build/run notes (incl. Windows layout and safe build).
- **Database:** Ent `Schema.Create` on startup (additive schema); `PRAGMA user_version` + optional file backup under `AppDir/backups/` and `internal/dbmanage` hooks when `DBSchemaUserVersion` is bumped for breaking changes / data moves.
- **Workspace (legacy) / Folder:** CRUD root folder via Wails + clean layering; duplicate **root** folder name rejected; UI modal + toast; row actions via **⋮** menu. *(Đã nâng cấp sang mô hình folder — xem Phase 2 và data-model.)*
- **Frontend:** Vite `base: './'`; layout hardened for WebView2; production CSS not via Tailwind CDN.

### Phase 1 - Core Request Runner

Scope:

- Request editor: method, URL, headers, query params, body.
- Body types: raw JSON/text, form-data (basic), x-www-form-urlencoded.
- Real HTTP execution in Go (`net/http`) with timeout.
- Response viewer: status, duration, size, headers, pretty JSON body.

Done when:

- Mock request flow is replaced by real backend HTTP execution.
- User can send real API requests end-to-end.

### Phase 2 - Folder and Request Management

Scope:

- **Một khái niệm `folders`:** `parent_id` NULL = folder gốc (hiển thị như list “workspace” cũ); lồng nhau qua `parent_id` → FK `folders.id`.
- **Requests** chỉ gắn **`folder_id`** (không còn `workspace_id` / `collection_id`).
- Tree sidebar: root folders → cây con (folder + request) đệ quy.
- CRUD folder + CRUD saved request; Save full request config (URL, method, headers, body, …).
- History: cột **`root_folder_id`** thay `workspace_id` — context sidebar khi Send.

Done when:

- Users can organize and reuse requests in a nested folder tree (tương đương Postman collections ở mức cơ bản).

### Phase 3 - History, Environments, Auth

Scope:

- Persist request history with request/response snapshot; **UI xem chi tiết** từng dòng (snapshot đã lưu).
- Environment variables (`{{var}}`) — bảng `environments` / `environment_variables` **global app** (xem [data-model-and-delivery-status.md](data-model-and-delivery-status.md)).
- Auth support: Bearer, Basic, API Key (header/query); cấu hình lưu trên saved request (`auth_json`).
- Variable resolution **trước** khi build HTTP request (sau đó merge auth).

Done when:

- Users can switch environments and authenticate requests efficiently.

**Delivered (close 2026-04-19):** đúng các mục trên trong code (Wails `EnvironmentHandler`, `internal/service/env_substitute.go`, `MergeAuthIntoHeadersAndQuery`, `RequestPanel` + modals, lịch sử resolved).

### Phase 4 - Productivity Features

Scope:

- ~~Multi-tab request editing.~~ **Done (2026-04-20)** — tabs lưu state riêng (URL/method/headers/body/form/multipart/auth/response/loading), **dirty dot** cho tab chưa lưu, **tab strip** có close + `+`, khôi phục tabs qua `localStorage` (`pmj.tabs.v1`), trần 20 tab; tái dùng tab rỗng khi Import cURL; khi "Save…" ad-hoc thành công → tab tự promote thành saved.
- ~~Search/filter for **folders** / requests and history.~~ **Done (2026-04-20)** — sidebar Folders có ô search (debounce 250ms) → gọi `SearchHandler.SearchTree` (SQLite `LIKE` case-insensitive cho `folders.name` + `requests.name`/`requests.url`), hiển thị flat list kèm breadcrumb path + highlight match; sidebar History có filter panel client-side (method multi-select, status group 2xx/3xx/4xx/5xx/other, URL substring, date range) + highlight URL.
- ~~Import/export project JSON (then evolve to Postman collection import).~~ **Import done (2026-04-20)** — Postman v2.1 / v2.0, OpenAPI 3.x (JSON + YAML), Insomnia v4; auto-detect format; map vào **folder tree mới** (root folder tự rename khi trùng); sibling trùng tên tự `" (n)"`; tùy chọn tạo Environment mới từ collection variables (optional activate). **Export Postman v2.1 done (2026-04-20)** — save dialog + file write từ root folder; export project JSON “native” vẫn tùy chọn.
- ~~Code snippet generation (curl/fetch).~~ **Done (2026-04-20)** — curl / fetch / axios / httpie; backend resolve `{{var}}` + auth như Execute.

Done when:

- Workflow is fast enough for daily API development.

**Delivered so far (Phase 4 partial — 2026-04-20):**

- **Item #1 — Import collection (Backend):** `internal/service/{collection_importer,postman_v21_importer,postman_v20_importer,openapi_importer,insomnia_importer}.go` + tests; usecase `internal/usecase/import_usecase.go` (persist tree + unique sibling name); Wails `delivery/ImportHandler` (`PickCollectionFile`, `PreviewCollectionFile`, `ImportCollectionFile`).
- **Item #1 — Import collection (Frontend):** `ImportCollectionModal.vue` (preview: format, counts, warnings, env opt-in) + nút **Import** trên sidebar Folders; refresh tree + auto-select root mới sau import.
- **Item #1 — Constraints/limits:** file ≤ `constant.MaxImportFileBytes` (25 MB); parser rejects Swagger 2.0 và file không nhận dạng được.
- **Item #2 — Multi-tab editor:** ref-store `frontend/src/stores/tabsStore.js` (`tabs[]`, `activeTabId`, actions `openSavedRequest` / `openBlank` / `openAdhocFromPayload` / `activateTab` / `closeTab` / per-tab response+loading setters, persist 200ms debounce vào `localStorage`); `RequestTabBar.vue` (tab strip, dirty dot dựa trên diff snapshot↔baseline, close + `+`); `RequestPanel.vue` expose `snapshot()` / `hydrate(snap)` + emit `snapshot-change` (debounced deep watch) / `baseline-committed` / `promote-to-saved`; `App.vue` rehydrate panel sau mỗi lần switch tab, response/loading thành computed theo active tab, per-tab setters đảm bảo response hạ cánh đúng tab kể cả khi user chuyển tab giữa chừng.
- **Item #3 — Search / filter (Backend):** `internal/repository/folder_repository.go` thêm `SearchByName(q, limit)` + `ListAllSkeleton()`; `internal/repository/request_repository.go` thêm `SearchByNameOrURL(q, limit)` (Ent `NameContainsFold` / `URLContainsFold` → SQLite `LIKE` case-insensitive, `LIMIT n+1` để báo truncate); usecase `internal/usecase/search_usecase.go` ghép 2 nguồn + build breadcrumb `path[]` từ skeleton (có cycle-safe), kèm unit test `TestPathForFolder`; Wails `delivery/SearchHandler.SearchTree(query, limit)` (empty query ⇒ empty result để UI fallback về cây).
- **Item #3 — Search / filter (Frontend):** `HighlightText.vue` (bôi orange match, không dùng regex — `indexOf` theo lowercase); `Sidebar.vue` thêm ô search ở tab Folders với debounce 250ms + anti-stale token, render flat results (folders + requests group) kèm path breadcrumb + highlight, click hit tự activate root + mở cây hoặc mở saved request; tab History thêm filter panel (method chip multi-select, status group 2xx/3xx/4xx/5xx/other, free-text URL/method/status, date-range `from`/`to` theo `<input type="date">`) client-side qua `computed filteredHistoryList`, highlight URL theo filter text, hiển thị counter `matched / total`.
- **Không đổi schema:** không bump `DBSchemaUserVersion` (tái sử dụng `folders` + `requests` + `environments` hiện có; tab state **chỉ lưu browser-side** qua `localStorage`; search dựa trên SQLite `LIKE`, không cần index mới ở scale local).
- **Flow B — Export Postman v2.1 (#4):** `internal/usecase/export_usecase.go` `ExportPostmanV21CollectionJSON`; Wails `ExportHandler.ExportPostmanV21` (save dialog + `os.WriteFile`); menu root ⋮ **Export Postman v2.1…** trong `Sidebar.vue`.
- **Flow B — Snippets (#5):** `internal/service/snippet.go` + `request_url.go` `FinalURLForRequest`; `SnippetHandler` (`RenderSnippet`, `ListSnippetKinds`); `SnippetPanel.vue` + `RequestPanel` `buildHttpExecutePayload` dùng chung với Send.
- **Flow B — DnD (#6.3):** repository `MoveToParent` / `MoveToFolder`; usecase `MoveFolder` / `MoveRequest`; Wails `FolderHandler.MoveFolder`, `SavedRequestHandler.MoveRequest`; `FolderTreeNode` drag/drop + `@tree-changed` → refresh mọi cây; hàng root trong `Sidebar` nhận drop (kéo folder/request vào collection) + `draggable` root.

**Bổ sung cùng ngày 2026-04-20 (polish + reorder):**

- **DnD “ra ngoài” / cùng cấp:** hàng folder chia vùng trên (~38%) = **Same level** (`MoveFolder` → parent của hàng đích), vùng dưới = **Inside**; hàng root: **Top-level** (`MoveFolder(…,'')`) vs **Inside** collection; highlight + tooltip.
- **Reorder thứ tự folder:** cột `folders.sort_order`, migrate **4→5**, backfill theo tên; `FolderHandler.ReorderFolder`; khe drop giữa các folder (nested + root) + append cuối danh sách.
- **UX click cây:** `@click` toggle trên toàn hàng (`role="button"` + padding), `items-stretch` + `min-h-[36px]`; khe reorder phía trên folder cũng toggle cùng folder — tránh “chỉ giữa hàng mới ăn click”.
- **Rename folder:** nested + root — **Rename** trong ⋮ (bỏ double-click folder); request vẫn có delayed click + double-click rename như trước.

### Phase 5 - Quality and Packaging

**Supported platforms (v1 official):** Windows x64. macOS / Linux là best-effort, không ký số, không chặn release.

Scope (4 nhóm đo được):

1. **Test bù lấp khoảng trống** — tập trung các layer đang 0 test:
   - `internal/dbmanage`: migration v4→v5 (backfill `sort_order`), `dropLegacyTablesForUUIDSchema`, `user_version` round-trip, `BackupDatabaseIfNonEmpty` (empty vs non-empty).
   - `internal/repository`: folder (unique `(parent_id,name)`, xoá đệ quy + null hoá FK history, `MoveToParent`, `ReorderFolderSibling`, `SearchByName` truncate), request (`CreateFull`/`GetByID` round-trip đủ headers/query/form/multipart/auth, `UpdateFull` replace, `SearchByNameOrURL`), environment (`SetActive` idempotent, `SaveVariables` replace, `ActiveVariableMap` chỉ enabled), history (`Save`/`ListSummaries` filter theo root).
   - `internal/usecase`: rule nghiệp vụ (folder name conflict, move vào subtree của chính mình bị chặn, request name conflict, env active duy nhất, duplicate variable key bị reject).
   - `internal/usecase` (import/export): **round-trip** Postman v2.1 — import → export → import lại → diff tree (folder + request name, method, URL, body).
2. **Smoke E2E ở tầng Go** (`internal/e2e` hoặc tương đương) — kịch bản đơn:
   create root folder → tạo env active với `{{base_url}}` → save request dùng `{{base_url}}` → `HTTPExecutor.Execute` bắn vào `httptest.Server` → assert history đúng → export Postman v2.1 → import lại → assert cây giống.
3. **CI tối thiểu** — GitHub Actions (`.github/workflows/ci.yml`): `go vet ./internal/...` + `go test ./internal/... -count=1` + `npm ci && npm run build` (không build Wails binary trong CI — tốn runner, chạy bằng `build-win-safe` thủ công trên Windows dev box).
4. **Release checklist + manual test plan** — hai tài liệu tick-được:
   - `.cursor/plans/release-checklist.md` — quality gate trước khi ghi nhận build là "internal release".
   - `.cursor/plans/manual-test-plan.md` — kịch bản tay cho tester, gom theo domain (folder, request, env, history, import/export, snippet, DnD/reorder, search).

Done when:

- `go test ./internal/... -count=1` pass trên local (Windows, CGO thường tắt) và `go test ./internal/... -count=1 -race` pass trên CI (Ubuntu có gcc sẵn).
- Smoke E2E pass.
- CI xanh.
- Release checklist tick đủ cho 1 bản Windows x64 build bằng `make build-win-safe`.
- Manual test plan đã được chạy 1 lần end-to-end trên Windows build đó.

**Ngoài scope (backlog giữ nguyên, không block):** Export project JSON native; migrate v2→v3 giữ dữ liệu; UI E2E (Playwright/WebView2); code signing Windows; notarize macOS.

### Phase 6 — Networking & Security (Corporate-ready)

**Why:** App hiện tại không cấu hình được proxy / CA custom → dev sau VPN hay SSL-inspection của cty không dùng được. Đây là điều kiện cần để app ra ngoài "dev box cá nhân".

Scope (3 nhóm):

1. **Proxy configuration**
   - Proxy mode: `none | system | manual`.
     - `system` → dùng `http.ProxyFromEnvironment` (đọc `HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY`).
     - `manual` → URL proxy (scheme + host + port, **không** embed user/pass trong URL) + ô **username** + **password** riêng (password lưu ciphertext `enc:v1:` trong `settings`) + `NO_PROXY` (comma-separated host; hỗ trợ suffix `.corp.net`).
   - Áp dụng: `HTTPExecutor` build `&http.Transport{Proxy: proxyFn, TLSClientConfig: ...}` 1 lần per request (không share global — để mỗi request có thể override sau này).
2. **Custom CA + TLS**
   - Danh sách file `.pem` / `.crt` do user import (qua Wails `OpenFileDialog`), lưu **nội dung** vào DB (bảng mới `trusted_cas`) để app portable (copy DB đi máy khác vẫn chạy).
   - Build `x509.CertPool`: bắt đầu từ `SystemCertPool()` + `AppendCertsFromPEM` cho từng CA user import. Validate PEM khi import, từ chối file không parse được.
   - **Insecure skip verify** — toggle **per-request** (không global), hiển thị badge đỏ trên tab và trong history detail để audit.
3. **Secret-type env variable + masking**
   - Bảng `environment_variables` thêm cột `kind` (`plain` | `secret`) — migration v5 → v6. Default `plain`, backfill tất cả dòng cũ là `plain`.
   - UI: variable secret hiện `••••••` mặc định, có nút 👁 để xem tạm; ô nhập password-type.
   - Redact: khi lưu history, detect giá trị của secret var có nằm trong URL/header/body/query/form → thay bằng `***`. Export Postman v2.1 collection → secret var **không xuất giá trị** (chỉ xuất key, value rỗng + description "secret").

DB migration:

- **DB v6**: `trusted_cas(id UUID, label TEXT, pem_content TEXT, enabled BOOL, created_at)` + `environment_variables.kind TEXT DEFAULT 'plain'` + `requests.insecure_skip_verify BOOL DEFAULT false` + `settings(key TEXT PK, value TEXT)` cho proxy mode/URL/username/password/no_proxy.
- `internal/dbmanage/data_migrate.go` thêm nhánh `v5→v6` (chỉ `ALTER TABLE ... ADD COLUMN` + `CREATE TABLE` — không destructive).

UI / Delivery:

- Panel **Settings** mới trong Sidebar (tab thứ 4 bên cạnh Folders / History / Environments), có 3 section: **Proxy**, **Custom CA**, **About**.
- Wails handler mới: `SettingsHandler` (`GetProxy`, `SetProxy`, `ListCACerts`, `AddCACert(label, pem)`, `ToggleCACert`, `RemoveCACert`).
- Per-request `InsecureSkipVerify` thêm vào `entity.HTTPExecuteInput` + `RequestPanel` toggle.

Done when:

- Request HTTPS qua proxy manual `http://user:pass@proxy:8080` vào 1 API bên ngoài thành công (manual test có checklist trong `manual-test-plan.md`).
- Import 1 file CA self-signed, gọi HTTPS tới `https://httpbin.local` self-signed — không còn lỗi `x509: signed by unknown authority`.
- Tạo env có biến `token` kind=secret, dùng trong `Authorization: Bearer {{token}}`, gửi → history không hiện giá trị thật của token.
- Export Postman v2.1 → mở JSON ra grep không thấy giá trị secret.
- `go test ./internal/... -race` xanh; CI xanh.
- Manual test plan có thêm section Phase 6 và đã chạy 1 lần.

Ngoài scope (đẩy backlog):

- OAuth2 flows (authorization code, client credentials) — chỉ bàn ở Phase sau.
- Client certificate auth (mTLS) — ít user cần ở v1.
- PAC file proxy.
- Encrypt at rest cho DB (v1 chấp nhận local plaintext).

### Phase 7 — UX Polish & Productivity

**Why:** Sau Phase 6 app đủ "dùng được trong cty", phase này biến app thành "dùng êm tay hằng ngày". Các việc đều nhỏ, low-risk, high-return, chủ yếu frontend.

Scope (6 hạng mục):

1. **Dashboard khi không có tab** + cho phép đóng tab cuối
   - Sửa `tabsStore.closeTab`: khi `tabs.length === 0` **không** auto-open blank. `App.vue` render `<DashboardHome />` thay vì `<RequestPanel />`.
   - Dashboard hiển thị: **Recent** (N=20 history gần nhất — đã có `HistoryRepository.ListSummaries`), **Quick actions** (New folder, Import collection, Import cURL, New env), **Stats** (tổng folder/request/env, DB size, lần backup gần nhất).
2. **Command palette (Ctrl/Cmd+K)**
   - Modal global, fuzzy search (client-side match — backend đã trả flat skeleton qua `ListAllSkeleton` + `SearchHandler.SearchTree`).
   - Groups: **Folders**, **Requests**, **Environments**, **Recent history**, **Commands** (New tab, New folder, Toggle env, Export, …).
   - Arrow keys + Enter; mở saved request / kích hoạt env / chạy command.
3. **Variable interpolation preview**
   - Dưới ô URL và body raw: 1 dòng text nhỏ "Resolved: `https://api.staging.corp/users/42`" — dùng `ActiveVariableMap` sẵn có, debounce 150ms. Ẩn nếu không có `{{var}}`.
4. **Duplicate folder / request**
   - Right-click trong sidebar tree hoặc menu ⋮ → **Duplicate**. Backend: `FolderUsecase.Duplicate(id)` / `RequestUsecase.Duplicate(id)` — tạo bản sao cùng parent, tự thêm `" (copy)"` với uniqueness theo rule hiện tại.
5. **Copy as cURL**
   - Nút "Copy as cURL" cạnh nút Send (hoặc trong menu ⋮ của request row). Tái sử dụng `internal/service/snippet.go` kind `curl` — đã có.
6. **Keyboard shortcuts cơ bản**
   - `Ctrl/Cmd+Enter` = Send; `Ctrl/Cmd+S` = Save; `Ctrl/Cmd+T` = New tab; `Ctrl/Cmd+W` = Close tab; `Ctrl/Cmd+Shift+E` = toggle environment menu; `Ctrl/Cmd+K` = palette; `Esc` = close modal hiện tại. Gom vào 1 composable `useKeyboardShortcuts()`.

Delivered (close 2026-04-26):

- Dashboard khi không có tab, quick actions, recent history, stats nhẹ; tab cuối đóng được và trạng thái rỗng được persist.
- In-app Help modal qua nút `?`, chứa keyboard shortcuts + productivity tips.
- Command palette `Ctrl/Cmd+K`: commands, folders/requests qua `SearchHandler.SearchTree`, environments, recent history.
- Variable preview dưới URL/body raw/XML, secret env values hiển thị `***`.
- Duplicate folder/request qua Wails handlers mới, copy recursive folder tree + saved request payload.
- Copy as cURL trên request panel, dùng lại `SnippetHandler` kind `curl_bash`.
- Shortcut: `Ctrl/Cmd+Enter`, `Ctrl/Cmd+S`, `Ctrl/Cmd+T`, `Ctrl/Cmd+W`, `Ctrl/Cmd+K`, `Ctrl/Cmd+Shift+E`, `Esc`.
- Vite code splitting: async components cho màn/modal ít dùng + `manualChunks` cho Vue / CodeMirror / formatter vendor; build không còn warning chunk > 500 kB.

Done when:

- [x] Đóng hết tab → thấy Dashboard; từ Dashboard click "Recent" mở lại được request cũ trong tab mới.
- [x] `Ctrl+K` → gõ 3 ký tự tên request → Enter → request mở trong tab.
- [x] URL `{{base_url}}/users/{{id}}` hiện preview resolved đúng.
- [x] Duplicate 1 folder 3-cấp-sâu → bản mới y hệt cấu trúc, không đụng bản gốc.
- [x] Copy as cURL paste vào terminal chạy giống app gửi.
- [x] Tất cả shortcut pass manual test section 7.
- [x] Không bump DB version.

Ngoài scope (backlog):

- Theme / dark mode toggle (đang follow system).
- Request diff viewer.
- Response body diff giữa 2 history row.

### Phase 8 — Collection Runner & Chaining

**Why:** Runner không có chaining thì yếu; chaining không có Runner thì chỉ phục vụ request đơn. Gộp lại thành 1 phase để đi đôi.

Scope (3 nhóm):

1. **Capture rules** (per saved request)
   - Bảng mới `request_captures(id UUID, request_id UUID FK, source, expression, target_var, scope, enabled, sort_order)`.
     - `source`: `response_body_json` | `response_body_regex` | `response_header` | `response_status`.
     - `expression`: JSONPath (vd `$.data.token`), regex (vd `"id":\\s*"(.*?)"` — dùng group 1), header name, hoặc để trống cho status.
     - `target_var`: tên env var đích (vd `auth_token`).
     - `scope`: `session` (reset khi đóng app — `SessionVariableStore` in-memory) | `environment` (ghi vào env active qua `SaveVariables`).
   - Sau mỗi lần `HTTPExecutor.Execute` thành công (Send bình thường hoặc Runner), usecase chạy capture theo `sort_order`, ghi kết quả về target.
   - UI: tab **Post-response** mới cạnh Params/Headers/Auth/Body trong `RequestPanel`, list captures với thêm/xoá/sort.
2. **Assertion rules** (per saved request)
   - Bảng `request_assertions(id, request_id, kind, expression, op, expected, enabled, sort_order)`.
     - `kind`: `status` | `header` | `response_json`.
     - `op`: `eq` | `neq` | `in` | `contains` | `exists` | `not_exists` | `match_regex`.
   - Chạy đồng thời với capture; kết quả (`pass/fail/error`) hiển thị trong **Response → Tests panel** (panel mới), và được Runner tổng hợp.
3. **Collection Runner**
   - Usecase `RunnerUsecase.RunFolder(folderID, envID, opts)` — gom toàn bộ request dưới folder (đệ quy, theo `sort_order`), iterate, apply env + capture + assertion, phát stream event qua Wails (`runtime.EventsEmit`) cho UI cập nhật bảng kết quả live.
   - Options: `stopOnFailure` | `iterations` (default 1, max 50) | `delayMs` | `timeoutPerRequest`.
   - UI: trang **Runner** (mở từ menu ⋮ của folder → "Run…" hoặc từ Dashboard), setup form + bảng kết quả (từng request: method, URL, status, duration, #assert pass/fail, nút "view body").
   - Export report: JSON đầy đủ + Markdown tóm tắt (paste vào PR).
   - Persist: bảng `runner_runs(id, folder_id, env_id, started_at, finished_at, summary_json)` + `runner_run_requests(id, run_id, request_id, ordinal, status_code, duration_ms, pass, fail, error_text, response_snippet)` để xem lại 10 run gần nhất.

DB migration:

- **DB v7**: `request_captures` + `request_assertions` + `runner_runs` + `runner_run_requests`. `internal/dbmanage/data_migrate.go` branch v6→v7 chỉ `CREATE TABLE` — additive.

Done when:

- Folder 5-request flow login → create → read → update → delete chạy xong với capture `$.token` → `{{auth_token}}` + assertion `status == 2xx`.
- Runner báo "5 passed / 0 failed" + download report Markdown.
- Stop-on-failure dừng đúng ở request lỗi.
- Capture scope=environment ghi vào env active, mở lại request đơn thấy `{{auth_token}}` resolve đúng giá trị mới.
- Capture + assertion vẫn hoạt động khi gửi request đơn (không qua Runner).
- Manual test plan có section 8 chạy 1 lần.
- `go test -race` + CI xanh.

Ngoài scope (backlog):

- Data-driven runner (CSV/JSON iteration) — đẩy Phase 10+.
- Parallel execution — mặc định tuần tự.
- Retry on failure — không làm.

### Phase 8 backlog (defer sang sau v1)

- Re-run từng request riêng lẻ ngay trong `RunnerRequestDetailModal` (re-use `HTTPExecutor` với payload đã resolve) — cải thiện vòng lặp debug post-run.
- Filter / search recent runs theo folder, env, status, ngày.
- Diff 2 run gần nhất (status / duration / response delta) — kiểu "regression check".
- Data-driven runner (CSV / JSON iteration), parallel execution, retry on failure (đã ghi nhận từ Phase 8.0).
- Cap riêng cho `runner_run_requests.response_body` khi persist (hiện đang phụ thuộc cap của HTTPExecutor) — chỉ làm khi đo được DB bloat trong thực tế.
- Bổ sung "Open Runner" trong Dashboard Quick Actions.

### Phase 9 — Scripting (Pre-request & Post-response) — **Done / closed 2026-04-30**

**Why:** Sau Phase 8 đã có capture + assertion tĩnh, nhưng một số use case thật yêu cầu logic động (ký HMAC, tính nonce, parse response phức tạp, loop retry thủ công). Script là đáp án cuối cùng + cho phép import script từ collection Postman chạy được.

Scope (4 nhóm):

1. **Engine + sandbox**
   - Embed **goja** (`github.com/dop251/goja`) — ES5.1+ subset, sync, well-maintained, Go-native.
   - Mỗi script chạy trong **goja.Runtime riêng**, có:
     - **Timeout**: `runtime.Interrupt("timeout")` sau `constant.ScriptTimeoutSeconds` (default 5s pre-req, 10s post-resp).
     - **No I/O**: không bind `fs`, `exec`, `net`; chặn `require()`.
     - **Console bridge**: `console.log/info/warn/error` → hàm Go thu vào slice, hiển thị trong **Console panel** trong Response.
2. **`pm.*` API subset (minimum viable để compat với script Postman cơ bản)**
   - `pm.environment.get(key)`, `pm.environment.set(key, value)`, `pm.environment.unset(key)`.
   - `pm.variables.get(key)` (resolve theo order: session → environment → collection vars placeholder).
   - `pm.request.url.toString()`, `pm.request.method`, `pm.request.headers.get(name)`, `pm.request.headers.add(...)`, `pm.request.body.raw`.
   - `pm.response.code`, `pm.response.headers.get(name)`, `pm.response.text()`, `pm.response.json()`, `pm.response.responseTime`.
   - `pm.test(name, fn)` — thêm record vào Tests panel (pass/fail), `pm.expect(actual).to.equal/be.ok/have.status/...` — mini chai (chỉ implement subset: `.to.equal`, `.to.eql`, `.to.be.true/false`, `.to.have.status`, `.to.include`, `.to.exist`, `.to.be.a('string')` — tương thích cú pháp nhưng reduced).
   - `pm.sendRequest(req, cb)` — **gọi lại `HTTPExecutor.Execute` synchronously** (goja sync, không có promise). API mô phỏng callback-style để tương thích, nhưng thực thi block.
3. **Persistence + editor**
   - Bảng `requests` thêm 2 cột: `pre_request_script TEXT DEFAULT ''`, `post_response_script TEXT DEFAULT ''`. Migration v8.
   - Import Postman v2.1: parse `event[]` với `listen: "prerequest" / "test"` → điền vào 2 cột này. Script sẽ chạy được ở mức `pm.*` subset; dùng API ngoài subset → log warning, không crash.
   - Export Postman v2.1: emit `event[]` ngược lại.
   - UI: trong `RequestPanel` — tab **Pre-request** + **Post-response** (đổi tên từ “Tests” Postman; CodeMirror JS, gợi ý `pmj` / `pm`), cùng strip **Captures** / **Assertions** (Phase 8). Console panel chung dưới Response.
4. **Integration với Runner (Phase 8)**
   - Runner chạy pre-request → request → post-response → capture → assertion. Pre-request fail (throw) → skip request, mark error. Post-response fail (throw) → mark fail, next request.
   - `pm.test()` kết quả được Runner gộp vào tổng pass/fail.

DB migration:

- **DB v9** (bump từ v8 — Phase 8.1 đã ở v8): `ALTER TABLE requests ADD COLUMN pre_request_script TEXT NOT NULL DEFAULT ''` + `ALTER TABLE requests ADD COLUMN post_response_script TEXT NOT NULL DEFAULT ''`. Migrate idempotent (helper `isDuplicateColumnErr` đã có sẵn).

Done when:

- Script `pm.environment.set('token', pm.response.json().token)` trong post-response chạy đúng, request sau dùng `{{token}}` thấy giá trị mới.
- Script loop 3 lần `pm.sendRequest` trong pre-request, log console, request chính vẫn gửi sau đó.
- Script vô hạn `while(true){}` → bị kill sau timeout, báo lỗi rõ, không treo app.
- Script thử `require('fs')` → throw `ReferenceError` hoặc bị block bởi sandbox, log rõ.
- Import 1 collection Postman có script `pm.test(...)` cơ bản → chạy được trong Runner, Tests panel hiện pass/fail đúng.
- Export lại collection → script text y nguyên.
- `go test ./internal/... -race` xanh; CI xanh (goja không cần CGO).
- Manual test plan có section 9 chạy 1 lần (§**L. Scripting** trong `manual-test-plan.md`).

**Delivered (as of close 2026-04-30):**

- `internal/service/pmj_runtime.go` + wiring `HTTPHandler.Execute` / Runner; `entity.HTTPExecuteInput` / `HTTPExecuteResult` script fields; `CloneSubstituteHTTPExecuteInput` substitute `{{var}}` trong script text.
- Ent + migrate **v9**; Postman import/export map `prerequest` / `test` ↔ DB columns.
- Frontend: `RequestPanel` script editors; `buildHttpExecutePayload` luôn gửi script khi có; `App.vue` watch đồng bộ store ↔ panel sau async load; `tabsStore` snapshot giữ `preRequestScript` / `postResponseScript`.
- **Release / QA:** tick checklist §Phase 9 trong `release-checklist.md` khi làm bản release có Scripting; chạy thủ công §L trước khi gọi internal release.

Ngoài scope (backlog **Phase 9.1** / sau v1):

- Full `pm.*` API (chai đầy đủ, cookies jar, visualizer).
- Async / Promise API.
- Script debugger / breakpoint.
- Shared library scripts (collection-level `pre-request` script).
- CommonJS / ES modules `require()`.

## Architecture Direction

- Keep clean layering:
  - Delivery (Wails handlers)
  - Usecase (business logic)
  - Repository (Ent + SQLite)
- Add a dedicated HTTP execution service.
- Prefer DTOs for Wails bridge instead of exposing DB entities directly.

## Immediate Backlog (Next Sprint)

Priority order (đồng bộ với checklist trong [data-model-and-delivery-status.md](data-model-and-delivery-status.md)):

1. ~~Complete Workspace UI CRUD UX (replace prompt/alert with proper modal + toast).~~ **Done (Phase 0).**
2. ~~Implement backend RequestExecutor service and response model.~~ **Done** (`internal/service/http_executor`, Wails `HTTPHandler`).
3. ~~Connect `RequestPanel` to real backend execution.~~ **Done**
4. ~~Persist history after each request.~~ **Done** (`histories` + sidebar History tab)
5. ~~**Folder + saved Request** CRUD + UI cây (nested folders + `folder_id`).~~ **Done (Phase 2, DB v3).**
6. ~~Environments + resolve `{{var}}` + auth (Bearer / Basic / API Key) theo Phase 3.~~ **Done (Phase 3).**

---

## Đề xuất bước tiếp theo (ưu tiên — cập nhật 2026-04-26)

Phase **5** đã **đóng** (quality gate baseline ổn định, CI xanh). **Phase 6** đã **đóng** (2026-04-21 — proxy + CA + secret env + insecure TLS + Settings UI). **Phase 7** đã **đóng** (2026-04-26 — UX polish/productivity + code splitting). **Phase 8** đã **đóng** (2026-04-26 — Collection Runner + capture/assertion + raw replay + iterations/delay/timeout, DB v6 → v7 → v8). **Phase 9** (Scripting) **đã đóng 2026-04-30** (DB v8 → v9).

1. ~~**Phase 6 — Networking & Security**~~ **Done (2026-04-21).**
   - Proxy (`none/system/manual`, URL + username + password ciphertext + `NO_PROXY`), custom CA (`trusted_cas`), per-request `insecure_skip_verify`, secret env vars + redact history/snippet payloads, Wails `SettingsHandler` + tab **Settings**.
   - DB **v6** (`trusted_cas`, `settings`, `environment_variables.kind`, `requests.insecure_skip_verify`).
2. ~~**Phase 7 — UX Polish & Productivity**~~ **Done (2026-04-26).**
   - Dashboard thay tab mặc định + cho đóng tab cuối; in-app Help `?`; Command palette Ctrl+K; variable preview; Duplicate folder/request; Copy as cURL; shortcuts cơ bản; Vite code splitting.
   - Không bump DB.
3. ~~**Phase 8 — Collection Runner & Chaining**~~ **Done (2026-04-26, closed in 8.0 + 8.1).**
   - 8.0: Capture rules (JSONPath/regex/header/status → env hoặc memory), Assertion rules (status/header/json-path/duration/size/regex, op eq/neq/contains/exists/...), Runner tuần tự folder theo `sort_order` + env active, persist run + per-request rows, stream Wails events, export report JSON/Markdown. **DB v6 → v7**.
   - 8.1: lưu raw resolved request/response cho từng runner row (5 cột mới trên `runner_run_requests`), modal xem chi tiết kế thừa UX của History; runner options đầy đủ: **Iterations** (≤50), **DelayMs** (≤60s, cancel-aware), **TimeoutPerRequestMs** (≤5 phút); HelpModal cập nhật mục Runner. **DB v7 → v8**.
4. ~~**Phase 9 — Scripting**~~ **Done (2026-04-30).** Pre-request + post-response; `pmj` / `pm` alias; Runner + Postman `event[]`; DB **v9**; script trên Send từ editor; hydrate tabs sau async `RequestPanel`.

**Nguyên tắc xuyên suốt các phase 6–9 (đã hoàn thành):**

- Mỗi phase đều mở rộng `manual-test-plan.md` (thêm section tương ứng) + `release-checklist.md` (thêm gate tương ứng).
- Mỗi phase có bump DB → thêm test migration trong `internal/dbmanage/data_migrate_test.go` theo pattern Phase 5 v4→v5.
- Mỗi phase có thay đổi `HTTPExecutor` hoặc thêm usecase lớn → bổ sung 1 case trong `internal/e2e/smoke_test.go` (hoặc file smoke riêng) để bắt regression end-to-end.

**Backlog chung (không thuộc Phase 6–9, giữ sau v1 hoặc sẽ gắn vào phase tương ứng sau):**

- Export project JSON native (đối xứng import).
- Migration v2 → v3 giữ dữ liệu (từ DB rất cũ).
- UI E2E (Playwright/WebView2).
- Code signing Windows; notarize macOS.
- OAuth2 flows (authorization code, client credentials); mTLS client cert; PAC proxy — cân nhắc gắn Phase 6.1 nếu nhu cầu thật nổi lên.
- Data-driven runner (CSV/JSON iteration); parallel run; retry on failure — cân nhắc gắn Phase 8.1.
- Full `pm.*` API (chai đầy đủ, cookies jar, visualizer), async/promise, script debugger, shared library scripts — cân nhắc gắn Phase 9.1.
- Theme/dark mode toggle; request diff; response body diff.
- Encrypt-at-rest cho DB (hiện plaintext local).
- **(minor, phát hiện qua smoke E2E 2026-04-21)** `ExportPostmanV21CollectionJSON` đi qua `url.Parse` + `url.Values.Encode()` khiến `{{var}}` trong URL bị percent-encode thành `%7B%7Bvar%7D%7D`. Postman desktop vẫn import lại được (sau giải mã); smoke test đã workaround bằng `url.PathUnescape` khi so sánh. Nên fix lúc chạm export path trong Phase 8/9 (khi thêm export script).

**Bảng hạng mục (lịch sử Phase 3–4 + backlog):**

| Thứ tự | Hạng mục | Lý do ngắn |
|--------|----------|------------|
| ~~**1**~~ | ~~History chi tiết (UI)~~ | **Done (Phase 3).** |
| ~~**2**~~ | ~~Environments + resolve `{{var}}`~~ | **Done (Phase 3).** |
| ~~**3**~~ | ~~Auth Bearer / Basic / API Key~~ | **Done (Phase 3).** |
| ~~**1**~~ | ~~**Import collection** (Postman/OpenAPI) — map vào **folder tree**~~ | **Done (Phase 4 item #1 — 2026-04-20).** |
| ~~**2**~~ | ~~**Multi-tab request editor**~~ | **Done (Phase 4 item #2 — 2026-04-20).** |
| ~~**3**~~ | ~~**Search / filter** (folders, requests, history)~~ | **Done (Phase 4 item #3 — 2026-04-20).** |
| ~~**4**~~ | ~~**Export** Postman v2.1~~ | **Done (2026-04-20)** — `ExportPostmanV21CollectionJSON` + Wails save dialog. |
| ~~**5**~~ | ~~**Snippet** curl/fetch/axios/httpie~~ | **Done (2026-04-20)** — `SnippetHandler` + `SnippetPanel`. |
| ~~**6**~~ | ~~**Polish cây folder** (expand, rename, DnD, reorder)~~ | **Done (2026-04-20)** — kể cả `sort_order`, vùng Same/Inside, full-row click. |
| **7** | **Migrate DB v2 → v3 giữ dữ liệu** (export/import) nếu cần | Backlog tùy chọn — chỉ khi cần upgrade không mất data từ DB rất cũ. |
| **8** | **Export project JSON native** (đối xứng Import) | Backlog tùy chọn — đã có Export Postman v2.1. |
| ~~**9**~~ | ~~**Quality gate baseline** (tests + smoke E2E + CI + release/manual docs)~~ | **Done (Phase 5 — 2026-04-21).** |
| ~~**10**~~ | ~~**Networking & Security** — proxy + custom CA + insecure skip verify + secret var~~ | **Done (Phase 6 — 2026-04-21).** |
| ~~**11**~~ | ~~**UX Polish & Productivity** — Dashboard, Ctrl+K palette, preview var, duplicate, copy cURL, shortcuts~~ | **Done (Phase 7 — 2026-04-26).** |
| ~~**12**~~ | ~~**Collection Runner & Chaining** — capture rules + assertion + Runner folder theo env + raw request/response replay + iterations/delay/timeout~~ | **Done (Phase 8 — 2026-04-26, DB v6 → v7 → v8).** |
| ~~**13**~~ | ~~**Scripting** — goja + `pmj`/`pm` subset pre-request & post-response~~ | **Done (Phase 9 — 2026-04-30, DB v8 → v9).** Tiếp theo: backlog **Phase 9.1** (full pm, async, debugger, collection-level script) trong section Phase 9 phía trên. |

**Gợi ý kỹ thuật:** mỗi hạng mục lớn — thêm/cập nhật test Go (`internal/service`, repository khi có logic); sau thay đổi Ent bump `DBSchemaUserVersion` + `data_migrate` nếu đổi DDL; mỗi phase có DB bump → viết thêm test migration theo pattern Phase 5 v4→v5; mỗi phase chạm HTTP layer hoặc thêm usecase lớn → bổ sung smoke E2E.
