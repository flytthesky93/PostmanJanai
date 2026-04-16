package constant

const (
	AppName      = "PostmanJanai"
	AppDbName    = "PostmanJanai.db"
	LogPath      = "logs/app.log"
	DebugLogPath = "logs/debug.log"

	// DBSchemaUserVersion — expected SQLite PRAGMA user_version after schema/data matches current Ent code.
	// When bumping: add a branch in dbmanage.migrateOneStep (data); backup runs before Schema.Create.
	DBSchemaUserVersion = 2

	// HTTPClientTimeout — total time for one request (including reading the response body).
	HTTPClientTimeoutSeconds = 60
	// HTTPMaxResponseBodyBytes — max response body read size (avoid OOM).
	HTTPMaxResponseBodyBytes = 10 << 20
)
