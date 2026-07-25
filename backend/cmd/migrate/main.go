package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"

	"github.com/Keviniscool-boy/supportflow/backend/internal/config"
	"github.com/Keviniscool-boy/supportflow/backend/internal/migrate"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	loadedConfig, err := config.Load()
	if err != nil {
		slog.Error("配置无效", "error", err)
		os.Exit(1)
	}
	if loadedConfig.DatabaseURL == "" {
		slog.Error("缺少 SUPPORTFLOW_DATABASE_URL")
		os.Exit(1)
	}
	database, err := sql.Open("pgx", loadedConfig.DatabaseURL)
	if err != nil {
		slog.Error("数据库连接初始化失败", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	if err := migrate.NewRunner(database).Up(context.Background()); err != nil {
		slog.Error("Migration 执行失败", "error", err)
		os.Exit(1)
	}
	slog.Info("Migration 执行完成")
}
