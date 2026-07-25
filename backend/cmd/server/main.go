package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Keviniscool-boy/supportflow/backend/internal/config"
	"github.com/Keviniscool-boy/supportflow/backend/internal/httpapi"
)

func main() {
	loadedConfig, err := config.Load()
	if err != nil {
		slog.Error("配置无效", "error", err)
		os.Exit(1)
	}
	server := httpapi.NewServer(loadedConfig)
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
