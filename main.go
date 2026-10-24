// news-service — временный микросервис новостей: берёт посты из VK API
// и отдаёт их фронту в нашем формате по GET /api/news.
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

	"news-service/internal/api"
	"news-service/internal/config"
	"news-service/internal/news"
	"news-service/internal/vk"
)

// main только собирает части вместе: конфиг → VK-клиент → хранилище →
// фоновое обновление → HTTP-сервер, и аккуратно всё останавливает.
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config error", "err", err)
		os.Exit(1)
	}

	// ctx отменится по Ctrl+C или по SIGTERM (так Docker останавливает контейнер).
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

	// Плавная остановка: дождаться текущих запросов, потом выйти.
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
