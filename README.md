# README

## About

This is the official Wails Vue template.

You can configure the project by editing `wails.json`. More information about the project settings can be found
here: https://wails.io/docs/reference/project-config

## Prerequisites

- Go 1.24+
- Node.js + npm
- Wails CLI v2

Install Wails CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Verify:

```bash
go version
node -v
npm -v
wails version
```

## Frontend notes

### Windows layout looks broken (narrow strip, huge empty area)

This is usually **not monitor size** (75" vs 27"). It happens when the embedded **WebView2** reports odd layout sizes / zoom, so HTML `%` widths collapse while the native window background shows on the side.

This project mitigates that by:

1. **No `Fullscreen: true`** — use **`WindowStartState: options.Maximised`** (see `main.go`).
2. **`frontend/index.html`** — `body { position: fixed; inset: 0 }` and `#app { position: absolute; inset: 0 }` so the page fills the WebView control (percent-height chains alone are unreliable).
3. **`windows.Options`**: `ZoomFactor: 1.0`, `DisablePinchZoom: true` so accidental zoom does not shrink the page viewport.
4. **No viewport `<meta>`** — desktop WebView + manual sizing above tends to be more reliable than `width=device-width` / fixed viewport tags.

- Do **not** add `https://cdn.tailwindcss.com` to `index.html`. The project uses Tailwind CSS v4 via Vite.
  Loading the CDN on top of the built stylesheet can break layout (for example the sidebar column).
- `frontend/vite.config.js` sets `base: './'` so JS/CSS load correctly inside the Wails embedded webview.
  Without relative asset URLs, `/assets/...` often fails and the UI looks broken (narrow strip, missing layout).
- `frontend/index.html` includes a small **critical `<style>` block** for `html/body/#app` sizing. The WebView does not always match a normal browser’s percent-height behaviour; the main app shell also uses **inline `position:fixed` + CSS grid** in `App.vue` so the layout works even if Tailwind CSS load is partial.

## Live Development

To run in live development mode, run `wails dev` in the project directory. This will run a Vite development
server that will provide very fast hot reload of your frontend changes. If you want to develop in a browser
and have access to your Go methods, there is also a dev server that runs on http://localhost:34115. Connect
to this in your browser, and you can call your Go code from devtools.

### Run without make (Windows friendly)

From project root:

```bash
go mod tidy
wails dev
```

## Building

To build a redistributable, production mode package, use `wails build`.

### Common build commands

```bash
# Default target (current OS)
wails build

# Windows
wails build -clean -platform windows/amd64 -o PostmanJanai

# Linux
wails build -clean -platform linux/amd64 -o PostmanJanai

# macOS Intel
wails build -clean -platform darwin/amd64 -o PostmanJanai

# macOS Apple Silicon
wails build -clean -platform darwin/arm64 -o PostmanJanai
```

Build output is generated in:

```bash
build/bin/
```

On Windows, the app binary is:

```bash
build/bin/PostmanJanai.exe
```

### Windows build error: `unlinkat ... PostmanJanai.exe: Access is denied`

This happens when `build/bin/PostmanJanai.exe` is still running or locked (often the app window is open). Wails uses `-clean` and must replace that file.

1. Quit the app and any dev session using the same exe.
2. Or terminate the process, then build again:

```powershell
taskkill /IM PostmanJanai.exe /F
wails build -clean -platform windows/amd64 -o PostmanJanai.exe
```

You can also run the helper script from the project root:

```powershell
.\build\win-safe.ps1
```

## App Data And Database Path

By default, application data is stored under `os.UserConfigDir()/PostmanJanai`.

On Windows, this is typically:

```bash
C:\Users\<User>\AppData\Roaming\PostmanJanai
```

Optional environment variables:

- `POSTMANJANAI_APP_DIR`: custom app workspace directory
- `POSTMANJANAI_DB_PATH`: custom sqlite file path (if not set, DB is `<APP_DIR>/PostmanJanai.db`)

Example (PowerShell):

```powershell
$env:POSTMANJANAI_APP_DIR = "E:\PostmanJanaiData"
$env:POSTMANJANAI_DB_PATH = "E:\PostmanJanaiData\db\main.db"
wails dev
```

## Logging

The app uses `slog` + `lumberjack` (file rotation enabled).

Log files:

- `logs/app.log`: business logs for app actions (`INFO/WARN/ERROR`)
- `logs/debug.log`: detailed diagnostic logs (request/repository traces)

Both files are created under `APP_DIR` (default or custom path above).

Current fixed rotation policy (hardcoded):

- `MaxSize`: 5 MB
- `MaxBackups`: 5
- `MaxAge`: 30 days
- `Compress`: true

