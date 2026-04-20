package constant

const (
	AppName      = "PostmanJanai"
	AppDbName    = "PostmanJanai.db"
	LogPath      = "logs/app.log"
	DebugLogPath = "logs/debug.log"

	// DBSchemaUserVersion — expected SQLite PRAGMA user_version after schema/data matches current Ent code.
	// When bumping: add a branch in dbmanage.migrateOneStep (data); backup runs before Schema.Create.
	DBSchemaUserVersion = 5

	// HTTPClientTimeout — total time for one request (including reading the response body).
	HTTPClientTimeoutSeconds = 60
	// HTTPMaxResponseBodyBytes — max response body read size (avoid OOM).
	HTTPMaxResponseBodyBytes = 10 << 20

	// MaxImportFileBytes — safety cap when importing Postman / OpenAPI / Insomnia files.
	// Large enough for realistic collections, small enough to reject accidental multi-GB inputs.
	MaxImportFileBytes = 25 << 20
)
