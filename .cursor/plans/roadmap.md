# PostmanJanai — Roadmap

**Spec DB + checklist triển khai kỹ thuật:** [data-model-and-delivery-status.md](data-model-and-delivery-status.md)

---

## Product Goal

Build a desktop API client (Postman-like) focused on:

- Reliable HTTP request execution.
- Workspace / Collection / Request management.
- Request history and debugging.
- Auth and environment variables.
- Smooth developer workflow for daily use.

## Phase completion log

| Phase | Status     | Notes (closing snapshot) |
|-------|------------|---------------------------|
| **0** | **Done**   | Closed **2026-04**: foundation stable for local-only app; see Phase 0 section below. |
| **1** | **Done**   | Closed **2026-04-17**: runner thật (`HTTPExecutor` + `HTTPHandler`), editor request/response/console, history persist + list, workspace CRUD + gắn `workspace_id` khi gửi, import request từ cURL. Chi tiết kỹ thuật: [data-model-and-delivery-status.md](data-model-and-delivery-status.md). |
| **2** | **Not started** | Collection / Request CRUD + cây sidebar — schema Ent đã có, chưa có usecase/repo/UI. |
| **3–5** | Not started | — |

---

### Phase 0 - Stabilize Foundation

Scope:

- Stabilize logging, app-data path, and DB migration behavior.
- Normalize frontend/backend data contracts (naming and payload shape).
- Add consistent UI-side error handling.

Done when:

- Workspace CRUD is stable.
- Logs are written reliably (`app.log` and `debug.log`).
- Build and run flow is documented and repeatable.

**Delivered (as of close):**

- **Logging & paths:** `slog` + rotating files (`app.log` / `debug.log`); `POSTMANJANAI_APP_DIR` / `POSTMANJANAI_DB_PATH`; README build/run notes (incl. Windows layout and safe build).
- **Database:** Ent `Schema.Create` on startup (additive schema); no versioned SQL migrations in-tree. `PRAGMA user_version` + optional file backup under `AppDir/backups/` and `internal/dbmanage` hooks when `DBSchemaUserVersion` is bumped for breaking changes / data moves.
- **Workspace:** CRUD via Wails + clean layering; duplicate workspace name rejected on create and update; UI uses modal + toast (no native `alert`/`prompt`), row actions via **⋮** menu.
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

### Phase 2 - Collection and Request Management

Scope:

- Add entities: `Collection`, `Request` with relation to `Workspace`.
- Tree sidebar: workspace -> collection -> request.
- CRUD for collections and requests.
- Save full request config (URL, method, headers, body, auth metadata).

Done when:

- Users can organize and reuse requests like basic Postman collections.

### Phase 3 - History, Environments, Auth

Scope:

- Persist request history with request/response snapshot *(phần persist + list đã có ở Phase 1; còn UI xem chi tiết từng dòng — xem backlog).*
- Environment variables (`{{var}}`) — **spec DB hiện tại:** bảng `environments` / `environment_variables` **global app** (xem [data-model-and-delivery-status.md](data-model-and-delivery-status.md)); có thể mở rộng gắn workspace sau nếu cần.
- Auth support: Bearer, Basic, API Key (header/query).
- Variable resolution before executing requests.

Done when:

- Users can switch environments and authenticate requests efficiently.

### Phase 4 - Productivity Features

Scope:

- Multi-tab request editing.
- Search/filter for collections and history.
- Import/export project JSON (then evolve to Postman collection import).
- Code snippet generation (curl/fetch).

Done when:

- Workflow is fast enough for daily API development.

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
5. Collection + Request **CRUD + UI cây** (schema Ent đã có; cần usecase/repo/UI).
6. Environments + resolve `{{var}}` + auth (Bearer / API Key) theo Phase 3.

---

## Đề xuất bước tiếp theo (ưu tiên — cập nhật 2026-04-17)

Dựa trên backlog trong [data-model-and-delivery-status.md](data-model-and-delivery-status.md) và kiến trúc hiện tại (`delivery` → `usecase`/`service`/`repository`):

| Thứ tự | Hạng mục | Lý do ngắn |
|--------|----------|------------|
| **1** | **Collection + Request:** repository, usecase, Wails handler, UI cây (workspace → collection → request), load/save cấu hình request | Mở **Phase 2**; cho phép gắn `request_id` vào `histories` khi gửi từ request đã lưu; là nền cho import collection sau. |
| **2** | **History chi tiết (UI):** click một dòng history → xem snapshot request/response đã lưu | Giá trị UX cao, ít phụ thuộc; bổ sung phần “xem lại” trong Phase 3 roadmap. |
| **3** | **Environments:** CRUD + một env active + resolve `{{var}}` trong pipeline trước `HTTPExecutor` | Theo spec DB global; cần trước auth phức tạp nếu token lấy từ biến. |
| **4** | **Auth:** Bearer / API Key (Basic sau) trên payload gửi đi | Sau khi có biến môi trường ổn định. |
| **5** | **Import collection** (Postman/OpenAPI) | Sau khi CRUD collection/request ổn định. |
| **6** | **Migrate int→UUID giữ dữ liệu** (nếu vẫn cần hỗ trợ user DB cũ) | Ưu tiên thấp hơn trừ khi có yêu cầu hỗ trợ; hiện path migrate là drop legacy. |

**Gợi ý kỹ thuật:** mỗi hạng mục lớn — thêm/cập nhật test Go (`internal/service`, repository khi có logic); sau thay đổi Ent bump `DBSchemaUserVersion` + `data_migrate` nếu đổi DDL.
