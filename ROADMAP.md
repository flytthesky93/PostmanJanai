# PostmanJanai Roadmap

## Product Goal

Build a desktop API client (Postman-like) focused on:

- Reliable HTTP request execution.
- Workspace / Collection / Request management.
- Request history and debugging.
- Auth and environment variables.
- Smooth developer workflow for daily use.

## Phase Plan

### Phase 0 - Stabilize Foundation

Scope:

- Stabilize logging, app-data path, and DB migration behavior.
- Normalize frontend/backend data contracts (naming and payload shape).
- Add consistent UI-side error handling.

Done when:

- Workspace CRUD is stable.
- Logs are written reliably (`app.log` and `debug.log`).
- Build and run flow is documented and repeatable.

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

- Persist request history with request/response snapshot.
- Environment variables (`{{base_url}}`) by workspace scope.
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

Priority order:

1. Complete Workspace UI CRUD UX (replace prompt/alert with proper modal + toast).
2. Implement backend RequestExecutor service and response model.
3. Connect `RequestPanel` to real backend execution.
4. Persist history after each request.
5. Add Collection + Request schema and basic CRUD.
6. Add basic auth modes (Bearer and API Key first).
