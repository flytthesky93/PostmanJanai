package constant

const (
	AppName      = "PostmanJanai"
	AppDbName    = "PostmanJanai.db"
	LogPath      = "logs/app.log"
	DebugLogPath = "logs/debug.log"

	// DBSchemaUserVersion — expected SQLite PRAGMA user_version after schema/data matches current Ent code.
	// When bumping: add a branch in dbmanage.migrateOneStep (data); backup runs before Schema.Create.
	DBSchemaUserVersion = 6

	// HTTPClientTimeout — total time for one request (including reading the response body).
	HTTPClientTimeoutSeconds = 60
	// HTTPMaxResponseBodyBytes — max response body read size (avoid OOM).
	HTTPMaxResponseBodyBytes = 10 << 20

	// MaxImportFileBytes — safety cap when importing Postman / OpenAPI / Insomnia files.
	// Large enough for realistic collections, small enough to reject accidental multi-GB inputs.
	MaxImportFileBytes = 25 << 20

	// ProxyTestTimeoutSeconds — timeout for "Test proxy" button (Phase 6).
	ProxyTestTimeoutSeconds = 15

	// EnvVarKindPlain / EnvVarKindSecret — environment_variables.kind values.
	EnvVarKindPlain  = "plain"
	EnvVarKindSecret = "secret"

	// ProxyMode* — settings.key = "proxy.mode" values.
	ProxyModeNone   = "none"
	ProxyModeSystem = "system"
	ProxyModeManual = "manual"

	// Setting keys (Phase 6).
	SettingKeyProxyMode     = "proxy.mode"
	SettingKeyProxyURL      = "proxy.url"
	SettingKeyProxyUser     = "proxy.username"
	SettingKeyProxyPassword = "proxy.password"
	SettingKeyProxyNoProxy  = "proxy.no_proxy"

	// SecretCipherPrefix — stored ciphertext prefix so legacy plaintext can coexist
	// and be recognised for future migration to an OS-keychain backed scheme.
	SecretCipherPrefix = "enc:v1:"
)
