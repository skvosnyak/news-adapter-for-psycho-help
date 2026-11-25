package news

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// fetchTimeout — на одну синхронизацию: вся стена VK — это несколько запросов.
const fetchTimeout = 2 * time.Minute

// Updater периодически забирает новости из Source, сохраняет их в файл
// и кладёт в Store. Файл — постоянная копия: с ним сервис отдаёт новости
// сразу после перезапуска и тогда, когда источник недоступен.
type Updater struct {
	source   Source
	store    *Store
	file     string        // путь к файлу с новостями; пусто — только память
	interval time.Duration // обычный интервал (например, 1h)
	retry    time.Duration // интервал после ошибки (например, 5m)
	log      *slog.Logger
}

// NewUpdater связывает источник, хранилище и файл.
func NewUpdater(
	src Source,
	store *Store,
	file string,
	interval,
	retry time.Duration,
	log *slog.Logger,
) *Updater {
	return &Updater{
		source:   src,
		store:    store,
		file:     file,
		interval: interval,
		retry:    retry,
		log:      log,
	}
}

// Run — бесконечный цикл обновления; запускается в отдельной горутине.
// Сначала новости берутся из файла. Если файл свежее interval, источник
// не дёргается до конца интервала — частые перезапуски не нагружают VK.
// Если источник не ответил, в Store остаются прежние новости,
// а следующая попытка будет раньше (retry).
// Цикл завершается, когда отменяют ctx (остановка сервиса).
func (u *Updater) Run(
	ctx context.Context,
) {
	wait := u.loadFile()
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}

		wait = u.interval
		err := u.refresh(ctx)
		u.store.SetSyncResult(time.Now().UTC(), err)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			u.log.Error("news refresh failed, serving saved news", "err", err, "retry_in", u.retry)
			wait = u.retry
		}
	}
}

// loadFile загружает новости из файла в Store и возвращает,
// сколько ждать до первого обращения к источнику.
func (u *Updater) loadFile() time.Duration {
	if u.file == "" {
		return 0
	}
	items, updatedAt, err := LoadFile(u.file)
	switch {
	case IsNotExist(err):
		u.log.Info("no saved news yet, waiting for first sync", "file", u.file)
		return 0
	case err != nil:
		u.log.Error("can't load saved news", "file", u.file, "err", err)
		return 0
	}

	u.store.ReplaceAt(items, updatedAt)
	age := time.Since(updatedAt)
	u.log.Info("news loaded from file", "file", u.file, "count", len(items),
		"updated_at", updatedAt, "age", age.Round(time.Second).String())
	return max(u.interval-age, 0)
}

// refresh делает одну попытку загрузки с таймаутом. Сначала новости
// записываются в файл и только потом в Store: если запись не удалась,
// память и файл не разойдутся.
func (u *Updater) refresh(
	ctx context.Context,
) error {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	items, err := u.source.Fetch(ctx)
	if err != nil {
		return err
	}
	// Пустой ответ без ошибки (стену закрыли, что-то сломалось в фильтрах) —
	// подозрительно: не затираем им сохранённые новости.
	if len(items) == 0 {
		return errors.New("source returned no news, keeping saved ones")
	}

	now := time.Now().UTC()
	if u.file != "" {
		if err := SaveFile(u.file, items, now); err != nil {
			return err
		}
	}
	u.store.ReplaceAt(items, now)
	u.log.Info("news refreshed", "count", len(items))
	return nil
}
