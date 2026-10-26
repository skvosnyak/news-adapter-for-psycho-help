.PHONY: project-up project-down \
		service-stop \
		logs-cleanup \
		store-cleanup

LOGS_DIR  := out/logs
STORE_DIR := store

project-up:
	@mkdir -p $(LOGS_DIR) $(STORE_DIR) && chmod 777 $(LOGS_DIR) $(STORE_DIR)
	@docker compose up -d

project-down:
	@docker compose down

service-stop:
	@docker compose stop news-app

logs-cleanup:
	@if [ -n "$$(docker compose ps -q --status running news-app)" ]; then \
		echo "news-app запущен, сначала: make service-stop" ; \
		exit 1; \
	fi
	@printf "Удалить все логи в $(LOGS_DIR)? [y/N] " ; \
	read ans; \
	if [ "$$ans" = "y" ]; then \
		find $(LOGS_DIR) -maxdepth 1 -type f -name '*.log' -delete && \
		echo "Логи удалены"; \
	else \
		echo "Удаление логов отменено"; \
	fi

store-cleanup:
	@if [ -n "$$(docker compose ps -q --status running news-app)" ]; then \
		echo "news-app запущен, сначала: make service-stop" ; \
		exit 1; \
	fi
	@printf "Удалить сохранённые новости в $(STORE_DIR)? [y/N] " ; \
	read ans ; \
	if [ "$$ans" = "y" ]; then \
		find $(STORE_DIR) -maxdepth 1 -type f -name '*.json*' -delete && \
		echo "Новости удалены"; \
	else \
		echo "Удаление новостей отменено"; \
	fi
