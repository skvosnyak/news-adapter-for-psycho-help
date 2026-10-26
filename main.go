// news-service — временный микросервис новостей: берёт посты из VK API
// и отдаёт их фронту в нашем формате по GET /api/news.
package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"news-service/internal/api"
	"news-service/internal/config"
	"news-service/internal/news"
	"news-service/internal/vk"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config error", "err", err)
		os.Exit(1)
	}

	if cfg.LogFile != "" {
		f, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			logger.Error("log file error", "err", err)
			os.Exit(1)
		}
		defer f.Close()
		logger = slog.New(slog.NewTextHandler(io.MultiWriter(os.Stdout, f), nil))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store := news.NewStore()
	source := vk.NewClient(cfg.VKToken, cfg.VKAPIVersion, cfg.VKGroup, cfg.VKPostsCount)
	updater := news.NewUpdater(source, store, cfg.RefreshInterval, cfg.RetryInterval, logger)
	go updater.Run(ctx)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           api.NewRouter(store, cfg.CORSOrigin),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	logger.Info("news-service started", "port", cfg.Port,
		"group", cfg.VKGroup, "refresh", cfg.RefreshInterval.String())

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server error", "err", err)
		os.Exit(1)
	}
	logger.Info("news-service stopped")
}
