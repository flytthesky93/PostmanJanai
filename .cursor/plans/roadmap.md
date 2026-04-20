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
| **4** | In progress | **Item #1 done (2026-04-20):** **Import collection** Postman v2.1 / v2.0 / OpenAPI 3.x (JSON+YAML) / Insomnia v4 → map vào folder tree + tạo Environment tùy chọn. Xem mục Phase 4 + backlog bên dưới. |
| **5** | Not started | — |

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

- Multi-tab request editing.
- Search/filter for **folders** / requests and history.
- ~~Import/export project JSON (then evolve to Postman collection import).~~ **Import done (2026-04-20)** — Postman v2.1 / v2.0, OpenAPI 3.x (JSON + YAML), Insomnia v4; auto-detect format; map vào **folder tree mới** (root folder tự rename khi trùng); sibling trùng tên tự `" (n)"`; tùy chọn tạo Environment mới từ collection variables (optional activate). Export project JSON còn pending.
- Code snippet generation (curl/fetch).

Done when:

- Workflow is fast enough for daily API development.

**Delivered so far (Phase 4 partial — 2026-04-20):**

- **Backend:** `internal/service/{collection_importer,postman_v21_importer,postman_v20_importer,openapi_importer,insomnia_importer}.go` + tests; usecase `internal/usecase/import_usecase.go` (persist tree + unique sibling name); Wails `delivery/ImportHandler` (`PickCollectionFile`, `PreviewCollectionFile`, `ImportCollectionFile`).
- **Frontend:** `ImportCollectionModal.vue` (preview: format, counts, warnings, env opt-in) + nút **Import** trên sidebar Folders; refresh tree + auto-select root mới sau import.
- **Constraints/limits:** file ≤ `constant.MaxImportFileBytes` (25 MB); parser rejects Swagger 2.0 và file không nhận dạng được.
- **Không đổi schema:** không bump `DBSchemaUserVersion` (tái sử dụng `folders` + `requests` + `environments` hiện có).

### Phase 5 - Quality and Packaging

Scope:

- Unit tests for critical usecase/repository logic.
- Basic E2E smoke tests for request flow.
- Release checklist and packaging hardening.

Done when:

- Stable internal release quality across supported platforms.

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

## Đề xuất bước tiếp theo (ưu tiên — cập nhật 2026-04-19)

Phase **3** đã đóng. Ưu tiên tiếp theo gắn **Phase 4** (productivity) và backlog kỹ thuật trong [data-model-and-delivery-status.md](data-model-and-delivery-status.md):

| Thứ tự | Hạng mục | Lý do ngắn |
|--------|----------|------------|
| ~~**1**~~ | ~~History chi tiết (UI)~~ | **Done (Phase 3).** |
| ~~**2**~~ | ~~Environments + resolve `{{var}}`~~ | **Done (Phase 3).** |
| ~~**3**~~ | ~~Auth Bearer / Basic / API Key~~ | **Done (Phase 3).** |
| ~~**1**~~ | ~~**Import collection** (Postman/OpenAPI) — map vào **folder tree**~~ | **Done (Phase 4 item #1 — 2026-04-20).** |
| **2** | **Migrate DB v2 → v3 giữ dữ liệu** (export/import) nếu cần | Hiện bump v3 **drop** domain — chỉ làm khi có yêu cầu upgrade không mất data. |
| **3** | **Polish UI folder tree:** expand/collapse, kéo-thả, đổi tên inline | UX nâng cao (Phase 4 / polish). |
| **4** | **Phase 4 khác:** multi-tab, search/filter, **export** project JSON, snippet curl/fetch | Theo mục Phase 4 trong roadmap. |

**Gợi ý kỹ thuật:** mỗi hạng mục lớn — thêm/cập nhật test Go (`internal/service`, repository khi có logic); sau thay đổi Ent bump `DBSchemaUserVersion` + `data_migrate` nếu đổi DDL.
