# Data model & trạng thái triển khai (PostmanJanai)

## Mục đích file này

- **Cấu trúc DB** (bảng, cột, FK, ERD) và **migration** / `PRAGMA user_version`.
- **Checklist kỹ thuật** đã / chưa làm — *Tiến độ đã triển khai* và *Todos* ở cuối file.

**Roadmap** (mục tiêu, phase 0–9, backlog): [roadmap.md](roadmap.md) (cùng thư mục `.cursor/plans/`).

> **Phase 6–9 (planned, 2026-04-21):**
> - **Phase 6 — Networking & Security:** proxy (system/manual) + custom CA pool + insecure skip verify (per-request) + secret-type env var. DB bump **v5 → v6** (`trusted_cas`, `settings`, `environment_variables.kind`).
> - **Phase 7 — UX Polish:** Dashboard khi không còn tab + cho đóng tab cuối, Ctrl+K palette, var preview, duplicate folder/request, Copy as cURL, shortcuts. **Không** bump DB.
> - **Phase 8 — Runner & Chaining:** capture rules (JSONPath/regex → env var) + assertion rules (status/header/json-path) + Collection Runner theo folder + env. DB bump **v6 → v7** (`request_captures`, `request_assertions`, `runner_runs`, `runner_run_requests`).
> - **Phase 9 — Scripting (bắt buộc):** goja + sandbox + `pm.*` subset cho pre-request & post-response, tích hợp Runner + import/export Postman script. DB bump **v7 → v8** (`requests.pre_request_script`, `requests.post_response_script`).
>
> Xem scope chi tiết + "Done when" trong từng section `Phase 6/7/8/9` của `roadmap.md`.

---

## Nguyên tắc

- **Bước 0 (bắt buộc):** thống nhất **cấu trúc bảng/cột/ràng buộc** dưới đây; chỉ sau khi bạn “sign-off” mới:
  - chỉnh `ent/schema/*.go` + `go generate`,
  - cập nhật `internal/entity`, repository/usecase,
  - nối UI / HTTP executor.
- **PK / FK:** dùng **UUID** (Ent `field.UUID`), không dùng `int` autoincrement cho bảng domain mới.
- **Thời gian:** `created_at` / `updated_at` — `datetime` (Ent `time.Time`).
- **Folder — tên không trùng trong cùng scope cha:** `UNIQUE (parent_id, name)`; **folder gốc** (`parent_id IS NULL`): không trùng tên giữa các root (enforce thêm ở usecase vì SQLite xử lý NULL trong UNIQUE).
- **Xóa folder:** đệ quy ở repository — xóa folder con trước, request trong từng folder, rồi folder; không còn bảng `workspaces` / `collections` tách biệt.
- **Environment sets:** scope **global app** (toàn bộ app dùng chung một bộ env; app desktop local-only).

### Thay đổi mô hình (2026-04 — **DB v3**)

- **Trước (v2):** `workspaces` → `collections` → `requests` (`workspace_id` + `collection_id` tùy chọn).
- **Sau (v3):** chỉ **`folders`** (cây tự tham chiếu `parent_id`) + **`requests.folder_id`** bắt buộc.
- **History:** `root_folder_id` (FK → folder gốc) thay cho `workspace_id` — ngữ nghĩa: folder đang chọn trên sidebar khi gửi (context lịch sử).

---

## Bảng và quan hệ (hiện tại trong code)

### `folders`

Thay thế **workspace + collection**: một bảng, cây lồng nhau.

| Cột | Kiểu | Ràng buộc |
|-----|------|-----------|
| `id` | UUID (TEXT) | PK |
| `parent_id` | UUID (TEXT) | NULL = **folder gốc** (hiển thị như hàng đầu sidebar); NOT NULL = con của folder cha |
| `name` | TEXT | NOT NULL |
| `description` | TEXT | NOT NULL, default `''` |
| `sort_order` | INTEGER | NOT NULL, default `0` — thứ tự hiển thị trong cùng parent (sidebar); reorder qua DnD / `ReorderFolder` (DB **v5**). |
| `created_at` | DATETIME | NOT NULL |

- **UNIQUE** (`parent_id`, `name`) — tên không trùng giữa các folder cùng cấp (cùng parent).
- **Edge Ent:** `parent` / `children` (self-reference), `requests`, `histories` (root context).

### `requests`

Mỗi request đã lưu thuộc **đúng một** folder.

| Cột | Kiểu | Ràng buộc |
|-----|------|-----------|
| `id` | UUID | PK |
| `folder_id` | UUID | NOT NULL, FK → `folders.id` |
| `name` | TEXT | NOT NULL |
| `method` | TEXT | NOT NULL, default `GET` |
| `url` | TEXT | NOT NULL |
| `body_mode` | TEXT | NOT NULL — enum logic app: `none`, `raw`, `xml`, `form_urlencoded`, `multipart`, … |
| `raw_body` | TEXT | NULL |
| `auth_json` | TEXT | NULL — JSON cấu hình auth (`none` / `bearer` / `basic` / `apikey`), optional |
| `created_at` | DATETIME | NOT NULL |
| `updated_at` | DATETIME | NOT NULL |

- **UNIQUE** (`folder_id`, `name`) — tên request không trùng trong cùng folder.

### `request_headers`

| Cột | Kiểu | Ràng buộc |
|-----|------|-----------|
| `id` | TEXT | PK, UUID |
| `request_id` | TEXT | NOT NULL, FK → `requests.id` CASCADE |
| `key` | TEXT | NOT NULL |
| `value` | TEXT | NOT NULL |
| `enabled` | BOOLEAN | NOT NULL, default true |
| `sort_order` | INTEGER | NOT NULL, default 0 |

- Index: (`request_id`, `sort_order`).

### `request_query_params`

| Cột | Kiểu | Ràng buộc |
|-----|------|-----------|
| `id` | TEXT | PK, UUID |
| `request_id` | TEXT | NOT NULL, FK → `requests.id` CASCADE |
| `key` | TEXT | NOT NULL |
| `value` | TEXT | NOT NULL |
| `enabled` | BOOLEAN | NOT NULL, default true |
| `sort_order` | INTEGER | NOT NULL, default 0 |

### `request_form_fields` (form-urlencoded & form-data tối giản)

| Cột | Kiểu | Ràng buộc |
|-----|------|-----------|
| `id` | TEXT | PK, UUID |
| `request_id` | TEXT | NOT NULL, FK → `requests.id` CASCADE |
| `field_kind` | TEXT | NOT NULL — `urlencoded` \| `multipart_text` \| `multipart_file` (theo code) |
| `key` | TEXT | NOT NULL |
| `value` | TEXT | NULL |
| `enabled` | BOOLEAN | NOT NULL, default true |
| `sort_order` | INTEGER | NOT NULL, default 0 |

### `histories` (khi **Send** HTTP)

**Quy tắc sản phẩm:**

- **Có** insert khi người dùng **Send** (kể cả lỗi transport / đọc body).
- **Không** ghi khi chỉ CRUD folder/request/env hoặc chưa gửi.

**Gắn entity:**

- `request_id` NULL = ad-hoc; NOT NULL = gửi từ saved request.
- `root_folder_id` NULL/optional = không gắn context folder gốc; NOT NULL = folder gốc đang chọn (sidebar) khi gửi — dùng để filter/gom history theo “space” đang làm việc.

| Cột | Kiểu | Ràng buộc |
|-----|------|-----------|
| `id` | TEXT | PK, UUID |
| `root_folder_id` | TEXT | NULL, FK → `folders.id` *(folder có `parent_id` NULL — ngữ nghĩa app; không enforce bằng CHECK trong schema tối thiểu)* |
| `request_id` | TEXT | NULL, FK → `requests.id` |
| `method` | TEXT | NOT NULL |
| `url` | TEXT | NOT NULL |
| `status_code` | INTEGER | NOT NULL |
| `duration_ms` | INTEGER | NULL |
| `response_size_bytes` | INTEGER | NULL |
| `request_headers_json` | TEXT | NULL |
| `response_headers_json` | TEXT | NULL |
| `request_body` | TEXT | NULL |
| `response_body` | TEXT | NULL |
| `created_at` | DATETIME | NOT NULL |

**Wails / UI:** payload gửi `root_folder_id` (thay `workspace_id`); optional `request_id` khi mở saved request.

### `environments` (global sets)

| Cột | Kiểu | Ràng buộc |
|-----|------|-----------|
| `id` | TEXT | PK, UUID |
| `name` | TEXT | NOT NULL, **UNIQUE** |
| `description` | TEXT | NOT NULL, default `''` |
| `is_active` | BOOLEAN | NOT NULL, default false |
| `created_at` | DATETIME | NOT NULL |
| `updated_at` | DATETIME | NOT NULL |

### `environment_variables`

| Cột | Kiểu | Ràng buộc |
|-----|------|-----------|
| `id` | TEXT | PK, UUID |
| `environment_id` | TEXT | NOT NULL, FK → `environments.id` CASCADE |
| `key` | TEXT | NOT NULL |
| `value` | TEXT | NOT NULL |
| `enabled` | BOOLEAN | NOT NULL, default true |
| `sort_order` | INTEGER | NOT NULL, default 0 |
| `created_at` | DATETIME | NOT NULL |
| `updated_at` | DATETIME | NOT NULL |

- **UNIQUE** (`environment_id`, `key`).

---

## Diễn giải ERD (tóm tắt)

```mermaid
erDiagram
  folders ||--o{ folders : parent_child
  folders ||--o{ requests : contains
  folders ||--o{ histories : rootContext
  requests ||--o{ request_headers : has
  requests ||--o{ request_query_params : has
  requests ||--o{ request_form_fields : has
  requests ||--o{ histories : optional
  environments ||--o{ environment_variables : has
```

---

## Migration & phiên bản DB

- **`PRAGMA user_version` hiện tại (code):** **`5`** (`internal/constant/app_constant.go` → `DBSchemaUserVersion`).
- **Luồng migrate:** backup DB (nếu non-empty) → `MigrateDataBetweenVersions` → `ent.Client.Schema.Create` → set `user_version`.
- **Các bước đã định nghĩa:**
  - `0 → 1`: placeholder.
  - `1 → 2`: drop bảng legacy (int PK) rồi recreate schema UUID (workspaces, collections, requests, …).
  - `2 → 3`: drop lại toàn bộ bảng domain có trong `dropLegacyTablesForUUIDSchema` (thêm `folders`), rồi `Schema.Create` — **mô hình mới folder + `requests.folder_id` + `histories.root_folder_id`**. **Không** có export/import tự động từ v2: dữ liệu cũ mất sau migrate (có file backup trong `AppDir/backups/` nếu backup chạy).
  - `3 → 4`: additive (Ent `Schema.Create`) — ví dụ `requests.auth_json`.
  - `4 → 5`: `ALTER TABLE folders ADD COLUMN sort_order …` + backfill theo tên trong `internal/dbmanage/data_migrate.go` (`backfillFolderSortOrder`).
- Nếu cần **giữ dữ liệu** khi nâng v2→v3: thêm bước export JSON / SQL trong `data_migrate` hoặc job sau `Schema.Create` (todo sản phẩm — **backlog**).

---

## Trạng thái (schema)

- **Schema Ent** khớp các bảng trên (folder, request, history, environment, …); **không** còn entity `workspace` / `collection` trong `ent/schema`.
- **Wails:** `FolderHandler` (gồm `MoveFolder`, **`ReorderFolder`**), `SavedRequestHandler` (gồm `MoveRequest`), `HTTPHandler`, `HistoryHandler`, **`EnvironmentHandler`**, `ImportHandler`, `SearchHandler`, **`ExportHandler`**, **`SnippetHandler`** — binding trong `frontend/wailsjs/`.

---

## DB & migration (ghi nhận kỹ thuật — cập nhật)

- **`histories`:** cột **`root_folder_id`** thay **`workspace_id`** từ DB v3.
- **HTTP execute:** DTO `root_folder_id` (JSON) thay `workspace_id`.
- **Saved request:** `SavedRequestFull.folder_id` duy nhất (bỏ `workspace_id` / `collection_id`).

---

## Tiến độ đã triển khai (cập nhật 2026-04-20)

- **Roadmap:** Phase **0–3** **đã đóng**; **Phase 4** **đã đóng** theo scope productivity (Import, Multi-tab, Search, export Postman v2.1, snippets, cây folder đầy đủ kể cả reorder + polish UX) — chi tiết [roadmap.md](roadmap.md).

### Nhật ký công việc 2026-04-20 (bổ sung trong ngày)

- **DnD:** vùng **Same level** / **Inside** trên hàng folder; root **Top-level** / **Inside**; highlight + strip gợi ý.
- **Reorder:** `folders.sort_order`, **DB v5**, `FolderHandler.ReorderFolder`, khe drop giữa folder (nested + root) và cuối list.
- **UX click:** toggle mở/thu trên **toàn hàng** (padding + `items-stretch`); khe reorder phía trên folder cũng gọi toggle; tránh dead zone mép trên/dưới.
- **Rename folder:** chỉ qua **⋮ → Rename** (nested + root); không double-click folder; request vẫn delayed single-click + double-click rename.

### Đã xong (Phase 1 + Phase 2)

- [x] Ent schema + generate — **v3: `folders`, `requests` + `folder_id`, `histories` + `root_folder_id`**
- [x] Migrate / backup — bump **2→3** (drop domain tables + recreate)
- [x] **HTTP executor** + Wails `HTTPHandler` (Execute, PickFile, ImportFromCurl)
- [x] **UI:** Request / Response / Console; History tab; **Folders** tab: root folders + **cây folder/request đệ quy**
- [x] **FolderHandler:** root CRUD + `ListChildFolders`
- [x] **SavedRequestHandler:** CRUD + `ListByFolder` + `Get`; RequestPanel load/save + `root_folder_id` / `request_id` khi Send
- [x] **Test data / repo hygiene** như trước

### Đã xong (Phase 3)

- [x] **`environments` / `environment_variables`:** Ent + repository + usecase + Wails **`EnvironmentHandler`** (CRUD env, CRUD biến, **một env active**)
- [x] **Substitute `{{var}}`:** `CloneSubstituteHTTPExecuteInput` + gọi từ `HTTPHandler.Execute` **trước** executor (URL, body, headers, query, form, multipart, trường auth)
- [x] **Auth:** `MergeAuthIntoHeadersAndQuery` — bearer / basic / apikey (header hoặc query); lưu **`auth_json`** trên `requests`
- [x] **History:** persist snapshot **đã resolve** (URL/body/headers như gửi thật)
- [x] **UI:** modal / flow **history chi tiết** (xem request/response đã lưu); editor **`{{var}}`** (chip, popover, caret nhảy khối trên CodeMirror + `EnvVarMirrorField`)

### Đã xong (Phase 4 — productivity)

- [x] **Multi-tab request editor** (2026-04-20):
  - **Store:** `frontend/src/stores/tabsStore.js` — reactive singleton (không dùng Pinia). Giữ `tabs: TabState[]` + `activeTabId`; mỗi `TabState` = `{ id, snapshot: RequestSnapshot, baseline: RequestSnapshot, response, loading }`.
  - **Actions:** `openSavedRequest(dto)` (nếu đã mở → activate + refresh snapshot, nếu chưa → tạo tab mới), `openBlank()`, `openAdhocFromPayload(curlPayload)` (tái dùng tab blank hiện tại nếu có), `activateTab(id)`, `closeTab(id)` (auto chọn tab kế cận; tự tạo blank khi đóng tab cuối), `updateActiveSnapshot(snap)`, `markActiveBaseline()`, `promoteActiveToSaved(dto)`, per-tab `setTabResponse(id,*)` / `setTabLoading(id,*)` (đảm bảo response hạ cánh đúng tab kể cả khi user switch tab giữa send).
  - **Persist:** key `pmj.tabs.v1` trong `localStorage`, debounce 200ms; lưu `tabs[i].{id,snapshot,baseline}` + `activeTabId`; **không** persist `response`/`loading` (transient). Trần `MAX_TABS = 20`.
  - **Dirty tracking:** `canonicalForDiff()` strip `activeTab` + `bodyRawEditor` (UI-only) → so sánh JSON giữa `snapshot` và `baseline`. `baseline` được commit khi: tab vừa tạo, mở saved request mới, save/update thành công (qua event `baseline-committed` / `promote-to-saved` từ RequestPanel).
  - **RequestPanel:** thêm `snapshot()` / `hydrate(snap)` qua `defineExpose`; watcher deep debounced 80ms trên toàn bộ reactive state → emit `snapshot-change`. Có cờ `hydrating` + `suppressSnapshotUpdate` (App.vue side) để không ghi đè baseline khi programmatically nạp lại. `saveSavedRequest` emit thêm `baseline-committed`; `submitSaveAdhoc` emit `promote-to-saved(created)`.
  - **UI:** `RequestTabBar.vue` — tab strip có method badge (color theo verb), dirty dot, close button (hover / middle-click), nút `+`, scroll ngang khi nhiều tab, active indicator cam.
  - **Persistence behaviour:** mở lại app → tabs khôi phục kèm nội dung form; **response bị xoá** (ý đồ: kết quả là ephemeral). Nếu user đang dirty, dirty dot vẫn hiện sau restart.
  - **Không đổi backend / schema:** hoàn toàn frontend-side; không thêm Wails handler, không bump `DBSchemaUserVersion`.

- [x] **Import collection** → folder tree (2026-04-20):
  - **Formats:** Postman Collection v2.1, Postman Collection v2.0 (legacy), OpenAPI 3.x (JSON + YAML), Insomnia v4 export (JSON). Auto-detect qua `internal/service/collection_importer.go` (probe JSON keys: `info.schema`, `openapi`, `_type: export`).
  - **Parsers (service):** `postman_v21_importer.go`, `postman_v20_importer.go`, `openapi_importer.go` (YAML → generic tree → JSON re-serialize để giữ `json.RawMessage`), `insomnia_importer.go` — mỗi file có bộ test `*_test.go` tương ứng.
  - **DTO trung gian:** `internal/entity/import_collection.go` — `ImportedCollection` / `ImportedItem` / `ImportedFolder` / `ImportedRequest` / `ImportedVariable` / `ImportOptions` / `ImportResult` (format-agnostic).
  - **Usecase:** `internal/usecase/import_usecase.go` — persist tree theo DFS, tạo **root folder mới luôn** (tên collection, auto rename khi trùng root), sibling trùng tên tự `" (n)"` qua `pickUniqueSiblingName`; tùy chọn tạo environment mới từ `variables` (optional activate).
  - **Delivery Wails:** `internal/delivery/import_handler.go` — `PickCollectionFile`, `PreviewCollectionFile`, `ImportCollectionFile`; wired trong `main.go` (OnStartup).
  - **Limits / errors:** cap file `constant.MaxImportFileBytes` (25 MB); error codes `IMP_701..IMP_707` trong `internal/constant/error_constant.go`.
  - **Frontend:** `frontend/src/components/ImportCollectionModal.vue` (preview tên, format, số folder/request, variables, warnings; option tạo + activate env) + nút **Import** trên sidebar Folders (`Sidebar.vue`); refresh folder tree + env list và toast sau khi import.
  - **DB impact (lúc triển khai):** tái sử dụng bảng `folders` / `requests` / …; các bump sau (`auth_json`, `sort_order`) xem mục **Migration & phiên bản DB** (hiện **v5**).

- [x] **Inline rename folder + saved request** (2026-04-20 — Phase 4 item #6.2; cập nhật UX cùng ngày):
  - **Folder:** chỉ **⋮ → Rename** → input inline (nested + root); **không** double-click folder; **Enter** / **Esc** / **blur** như trước.
  - **Request:** double-click → input inline; single-click mở tab (delayed ~220ms để tách double-click).
  - Menu ⋮: folder **Rename** (inline) + **Edit folder…** (modal mô tả); request **Rename** dùng inline.
  - `emit('saved-request')` từ tree khi đổi tên request → `Sidebar` forward → `App` `onSavedRequestUpdated` (refresh catalog).
  - **Không đổi backend / schema** (dùng `FolderHandler.UpdateFolder` / `SavedRequestHandler.Update` hiện có).

- [x] **Persist folder tree expand/collapse** (2026-04-20 — Phase 4 item #6.1):
  - `Sidebar.vue` sở hữu `expandedFolderIds` (plain `Record<string, boolean>`) và `rootTreeExpanded`; provide `expandedFolderIds` để mọi `FolderTreeNode` share một trạng thái thay vì mỗi instance tự giữ.
  - Persist vào `localStorage`: `pmj.sidebar.folder-expanded.v1` (nested) và `pmj.sidebar.root-expanded.v1` (root) — debounce 200ms, load trước khi render cây để tránh flicker.
  - Stale ids sau khi xoá folder là vô hại (id không tồn tại chỉ ẩn mục).
  - **Không đổi backend / schema.**

- [x] **Search / filter** (folders, saved requests, history) (2026-04-20):
  - **Backend:** `internal/repository/folder_repository.go` — `SearchByName(ctx, query, limit) ([]*FolderItem, truncated, error)` dùng `folder.NameContainsFold` + `LIMIT n+1` để phát hiện truncate; thêm `ListAllSkeleton(ctx)` (chỉ `id` / `name` / `parent_id`) để tính breadcrumb. `internal/repository/request_repository.go` — `SearchByNameOrURL(ctx, query, limit)` OR `name`/`url` với `ContainsFold`.
  - **Usecase:** `internal/usecase/search_usecase.go` — `SearchTree(query, limit)` ghép folder + request hits, build `path[]` (breadcrumb tên folders từ root) từ skeleton, cycle-safe. Test `search_usecase_test.go` cho `pathForFolder` (root / mid / leaf / sibling / missing / cycle).
  - **Delivery:** `internal/delivery/search_handler.go` — Wails `SearchHandler.SearchTree(query, limit)`; empty query ⇒ empty result để UI fallback về cây thường. Wire trong `main.go`.
  - **Frontend:** `frontend/src/components/HighlightText.vue` (utility bôi match không regex); `Sidebar.vue` bổ sung:
    - Tab **Folders**: ô search đầu panel, debounce 250ms + anti-stale token, render flat list "Folders · n" + "Requests · n" kèm breadcrumb path + highlight; click folder hit ⇒ activate root + expand, click request hit ⇒ `open-saved-request`.
    - Tab **History**: input free-text (URL/method/status substring, highlight trên URL) + filter panel toggle (method chip multi-select, status group 2xx/3xx/4xx/5xx/other, date range `from`/`to`), pure client-side qua `computed filteredHistoryList`; counter `matched / total` + "Clear all".
  - **Giới hạn:** mặc định 100 hit/nhóm, cứng ≤ 500 (`searchMaxLimit`), truncate hiển thị hint cho user.
  - **DB impact:** **không** đổi schema — dùng SQLite `LIKE` qua Ent `*ContainsFold`; ở scale local (thousands) không cần FTS/index phụ.

### Chưa làm / backlog (sau Phase 4)

- [ ] **Export** project JSON “native” / đối xứng đầy đủ Import (tùy chọn — đã có **Postman Collection v2.1** từ root folder)
- [ ] **Migrate v2→v3 giữ dữ liệu** (nếu cần) — hiện path bump cũ là **drop** + backup

### Đã xong (Flow B + polish — 2026-04-20)

- [x] **Export Postman Collection v2.1** — `export_usecase.go` + `ExportHandler.ExportPostmanV21`; UI save dialog từ menu root folder
- [x] **Snippet** curl / fetch / axios / httpie — `SnippetHandler` + resolve env + auth (cùng pipeline Execute)
- [x] **Polish cây folder** — expand/collapse `localStorage`; rename folder qua ⋮; DnD `MoveFolder` / `MoveRequest`; vùng Same/Inside; **reorder** `sort_order` + `ReorderFolder` + **DB v5**; full-row click + khe reorder nhận toggle

---

# Todos (checklist)

- [x] Sign-off schema (folder + request — đã triển khai v3)
- [x] Ent schema + generate (`folders`, cập nhật `requests` / `histories`)
- [x] Migrate + backup (bump v3; drop — chưa migrate giữ dữ liệu v2)
- [x] HTTP executor & UI (core)
- [x] History: persist + list + `root_folder_id` từ UI
- [x] Import request từ cURL
- [x] **Folder + saved request:** repository, usecase, Wails, UI cây
- [x] **Environments** + **environment_variables** (usecase + UI + `EnvironmentHandler`)
- [x] **Active env duy nhất** + **resolve `{{var}}`** trước gửi request
- [x] Import **collection** (file) vào folder tree — Postman v2.1/v2.0, OpenAPI 3.x (JSON+YAML), Insomnia v4 (2026-04-20)
- [x] **Multi-tab** request editor + persist `localStorage` (2026-04-20)
- [x] **Search / filter** folder + saved request + history (2026-04-20)
- [x] **Export** Postman Collection v2.1 (file) — 2026-04-20
- [x] **Snippet** curl / fetch / axios / httpie — 2026-04-20
- [x] **DnD** move folder/request + **reorder** + **DB v5** `sort_order` — 2026-04-20
- [ ] (Tùy chọn) **Export/import** khi nâng DB v2→v3 để không mất data

---

## Đề xuất bước tiếp theo

**Phase 5** (quality, test, packaging) — **đang triển khai (2026-04-21)**; scope chi tiết + DoD xem [roadmap.md §Phase 5](roadmap.md). 4 nhóm đo được: (1) test bù lấp `dbmanage` / `repository` / `usecase` / import-export round-trip; (2) smoke E2E Go (httptest + ent SQLite tạm); (3) CI tối thiểu (`.github/workflows/ci.yml`); (4) [release-checklist.md](release-checklist.md) + [manual-test-plan.md](manual-test-plan.md). Windows x64 là platform chính thức v1; macOS/Linux best-effort, unsigned.

**Backlog ngoài Phase 5:** export project JSON native; migration v2→v3 giữ dữ liệu; UI E2E (Playwright); code signing Windows; notarize macOS.
