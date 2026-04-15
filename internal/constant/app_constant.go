package constant

const (
	AppName      = "PostmanJanai"
	AppDbName    = "PostmanJanai.db"
	LogPath      = "logs/app.log"
	DebugLogPath = "logs/debug.log"

	// DBSchemaUserVersion — SQLite PRAGMA user_version mong đợi sau khi schema/data đã khớp code Ent hiện tại.
	// Khi tăng số này: thêm nhánh trong dbmanage.migrateOneStep (data) và backup sẽ chạy trước khi Schema.Create.
	DBSchemaUserVersion = 1
)
