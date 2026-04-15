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

