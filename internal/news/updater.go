package news

import (
	"context"
	"log/slog"
	"time"
)

// Updater периодически забирает новости из Source и кладёт их в Store.
type Updater struct {
	source   Source
	store    *Store
	interval time.Duration // обычный интервал (например, 1h)
	retry    time.Duration // интервал после ошибки (например, 5m)
	log      *slog.Logger
}

// NewUpdater связывает источник и хранилище.
func NewUpdater(src Source, store *Store, interval, retry time.Duration, log *slog.Logger) *Updater {
	return &Updater{source: src, store: store, interval: interval, retry: retry, log: log}
}

// Run — бесконечный цикл обновления; запускается в отдельной горутине.
// Первая загрузка происходит сразу при старте. Если VK не ответил,
// старые новости остаются в Store, а следующая попытка будет раньше (retry).
// Цикл завершается, когда отменяют ctx (остановка сервиса).
func (u *Updater) Run(ctx context.Context) {
	for {
		wait := u.interval
		if err := u.refresh(ctx); err != nil {
			u.log.Error("news refresh failed", "err", err, "retry_in", u.retry)
			wait = u.retry
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

// refresh делает одну попытку загрузки с таймаутом.
func (u *Updater) refresh(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	items, err := u.source.Fetch(ctx)
	if err != nil {
		return err
	}
	u.store.Replace(items)
	u.log.Info("news refreshed", "count", len(items))
	return nil
}
