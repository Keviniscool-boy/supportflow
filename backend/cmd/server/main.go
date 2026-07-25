package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Keviniscool-boy/supportflow/backend/internal/config"
	"github.com/Keviniscool-boy/supportflow/backend/internal/httpapi"
	"github.com/Keviniscool-boy/supportflow/backend/internal/session"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	loadedConfig, err := config.Load()
	if err != nil {
		slog.Error("配置无效", "error", err)
		os.Exit(1)
	}
	var sessionStore session.Store = session.NewMemoryStore(loadedConfig.SessionTTL)
	var database *sql.DB
	if loadedConfig.DatabaseURL != "" {
		database, err = sql.Open("pgx", loadedConfig.DatabaseURL)
		if err != nil {
			slog.Error("数据库连接初始化失败", "error", err)
			os.Exit(1)
		}
		pingContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = database.PingContext(pingContext)
		cancel()
		if err != nil {
			slog.Error("数据库不可用", "error", err)
			_ = database.Close()
			os.Exit(1)
		}
		sessionStore = session.NewSQLStore(database, loadedConfig.SessionTTL)
	}
	if database != nil {
		defer database.Close()
	}
	server := httpapi.NewServerWithSessionStore(loadedConfig, sessionStore)
	httpServer := &http.Server{
		Addr:              loadedConfig.HTTPAddress,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       loadedConfig.RequestTimeout,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	shutdownContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverError := make(chan error, 1)
	slog.Info("SupportFlow 服务启动", "address", loadedConfig.HTTPAddress, "environment", loadedConfig.Environment)
	go func() {
		serverError <- httpServer.ListenAndServe()
	}()
	select {
	case <-shutdownContext.Done():
		gracefulContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(gracefulContext); err != nil {
			slog.Error("服务优雅停止失败", "error", err)
			os.Exit(1)
		}
		slog.Info("SupportFlow 服务已停止")
	case err := <-serverError:
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		slog.Error("服务停止", "error", err)
		os.Exit(1)
	}
}
