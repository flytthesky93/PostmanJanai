package main

import (
	"PostmanJanai/ent"
	"PostmanJanai/internal/config"
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/delivery"
	"PostmanJanai/internal/pkg/logger"
	"PostmanJanai/internal/repository"
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
	drv := entsql.OpenDB(dialect.SQLite, db)

	// Khởi tạo Ent Client
	client := ent.NewClient(ent.Driver(drv))
	defer func(client *ent.Client) {
		err := client.Close()
		if err != nil {
			log.Fatalf("failed closing client: %v", err)
		}
	}(client)

	// Chạy auto migration (Tự động tạo/sửa bảng)
	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	// 3. Khởi tạo các lớp theo Clean Architecture
	//Repository
	//historyRepo := repository.NewHistoryRepository(client)
	workspaceRepo := repository.NewWorkspaceRepository(client)
	// Usecase
	workspaceUc := usecase.NewWorkspaceUsecase(workspaceRepo)
	// Handler
	appHandler := delivery.NewAppHandler() // Truyền uc vào đây nếu cần
	workspaceHandler := delivery.NewWorkspaceHandler(workspaceUc)

	// Create application with options
	// Fullscreen:true often breaks WebView2 layout/clientsize on Windows (narrow column / blank area).
	// Use a normal window, start maximised, and enforce a sane minimum size.
	err = wails.Run(&options.App{
		Title:             constant.AppName,
		Width:             1280,
		Height:            800,
		MinWidth:          900,
		MinHeight:         600,
		Fullscreen:        false,
		WindowStartState:  options.Maximised,
		DisableResize: false,
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
			workspaceHandler.SetContext(ctx)
		},
		Bind: []interface{}{
			appHandler,
			workspaceHandler,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
