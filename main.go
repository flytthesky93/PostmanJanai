package main

import (
	"PostmanJanai/ent"
	"PostmanJanai/internal/config"
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/dbmanage"
	"PostmanJanai/internal/delivery"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/repository"
	"PostmanJanai/internal/service"
	"PostmanJanai/internal/usecase"
	"context"
	"database/sql"
	"embed"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"log"

	_ "github.com/glebarez/go-sqlite"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	// 1. Load Config (Tạo folder app)
	config.LoadConfig()
	cfg := config.GetConfig()
	logger.Init(cfg.AppDir)
	logger.L().Info("Application started", "app_dir", cfg.AppDir, "db_path", cfg.DbPath)
}

func main() {

	cfg := config.GetConfig()

	db, err := sql.Open("sqlite", cfg.DbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}

	ctx := context.Background()
	ver, err := dbmanage.UserVersion(db)
	if err != nil {
		log.Fatalf("failed reading PRAGMA user_version: %v", err)
	}
	target := constant.DBSchemaUserVersion
	if ver < target {
		if _, err := dbmanage.BackupDatabaseIfNonEmpty(cfg.DbPath, cfg.AppDir); err != nil {
			log.Fatalf("database backup: %v", err)
		}
		if err := dbmanage.MigrateDataBetweenVersions(ctx, db, ver, target); err != nil {
			log.Fatalf("data migration: %v", err)
		}
	}

	drv := entsql.OpenDB(dialect.SQLite, db)

	client := ent.NewClient(ent.Driver(drv))
	defer func(client *ent.Client) {
		err := client.Close()
		if err != nil {
			log.Fatalf("failed closing ent client: %v", err)
		}
	}(client)

	if err := client.Schema.Create(ctx); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}
	if err := dbmanage.SetUserVersion(db, target); err != nil {
		log.Fatalf("failed setting PRAGMA user_version: %v", err)
	}

	folderRepo := repository.NewFolderRepository(client)
	savedRequestRepo := repository.NewRequestRepository(client)
	historyRepo := repository.NewHistoryRepository(client)
	requestRuleRepo := repository.NewRequestRuleRepository(client)
	runnerRepo := repository.NewRunnerRepository(client)

	secretCipher, err := service.NewSecretCipher()
	if err != nil {
		log.Fatalf("secret cipher: %v", err)
	}
	settingsRepo := repository.NewSettingsRepository(client)
	trustedCARepo := repository.NewTrustedCARepository(client)
	envRepo := repository.NewEnvironmentRepository(client, secretCipher)

	folderUc := usecase.NewFolderUsecaseWithRequests(folderRepo, savedRequestRepo)
	savedRequestUc := usecase.NewRequestUsecase(folderRepo, savedRequestRepo)
	envUc := usecase.NewEnvironmentUsecase(envRepo)
	importUc := usecase.NewImportUsecase(folderRepo, savedRequestRepo, envRepo)
	searchUc := usecase.NewSearchUsecase(folderRepo, savedRequestRepo)
	exportUc := usecase.NewExportUsecase(folderRepo, savedRequestRepo)
	settingsUc := usecase.NewSettingsUsecase(settingsRepo, trustedCARepo, secretCipher)
	tf := &service.HTTPTransportFactory{Settings: settingsRepo, CAs: trustedCARepo, Cipher: secretCipher}
	httpExecutor := service.NewHTTPExecutor(tf)
	runnerUc := usecase.NewRunnerUsecase(folderRepo, savedRequestRepo, requestRuleRepo, envRepo, runnerRepo, httpExecutor)

	appHandler := delivery.NewAppHandler()
	folderHandler := delivery.NewFolderHandler(folderUc)
	savedRequestHandler := delivery.NewSavedRequestHandler(savedRequestUc)
	historyHandler := delivery.NewHistoryHandler(historyRepo)
	environmentHandler := delivery.NewEnvironmentHandler(envUc)
	importHandler := delivery.NewImportHandler(importUc)
	searchHandler := delivery.NewSearchHandler(searchUc)
	exportHandler := delivery.NewExportHandler(exportUc)
	settingsHandler := delivery.NewSettingsHandler(settingsUc)
	httpHandler := delivery.NewHTTPHandler(httpExecutor, historyRepo, envRepo, savedRequestRepo, requestRuleRepo)
	snippetHandler := delivery.NewSnippetHandler(envRepo)
	ruleHandler := delivery.NewRuleHandler(requestRuleRepo)
	runnerHandler := delivery.NewRunnerHandler(runnerUc)

	err = wails.Run(&options.App{
		Title:            constant.AppName,
		Width:            1280,
		Height:           800,
		MinWidth:         900,
		MinHeight:        600,
		Fullscreen:       false,
		WindowStartState: options.Maximised,
		DisableResize:    false,
		Windows: &windows.Options{
			DisablePinchZoom:     true,
			IsZoomControlEnabled: false,
			ZoomFactor:           1.0,
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup: func(ctx context.Context) {
			appHandler.SetContext(ctx)
			folderHandler.SetContext(ctx)
			savedRequestHandler.SetContext(ctx)
			historyHandler.SetContext(ctx)
			environmentHandler.SetContext(ctx)
			importHandler.SetContext(ctx)
			searchHandler.SetContext(ctx)
			exportHandler.SetContext(ctx)
			settingsHandler.SetContext(ctx)
			httpHandler.SetContext(ctx)
			snippetHandler.SetContext(ctx)
			ruleHandler.SetContext(ctx)
			runnerHandler.SetContext(ctx)
		},
		Bind: []interface{}{
			appHandler,
			folderHandler,
			savedRequestHandler,
			historyHandler,
			environmentHandler,
			importHandler,
			searchHandler,
			exportHandler,
			settingsHandler,
			httpHandler,
			snippetHandler,
			ruleHandler,
			runnerHandler,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
