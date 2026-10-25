.PHONY: project-up project-down \
		logs-cleanup logs-prune 

LOGS_DIR := out/logs

project-up:
	@mkdir -p $(LOGS_DIR) && chmod 777 $(LOGS_DIR)
	@docker compose up -d

project-down:
	@docker compose down 

logs-cleanup:
	@if [ -n "$$(docker compose ps -q --status running news-app )" ]; then \
		echo "news-app запущен, сначала: make service-stop" ; \
		exit 1; \
	fi
	@printf "Удалить все логи в $(LOGS_DIR)? [y/N] " ; \
	read ans; \
	if [ "$$ans" = "y" ]; then \
		find $(LOGS_DIR) -maxdepth 1 -type f -name '*.log'  -delete && \
		echo "Логи удалены"; \
	else \
		echo "Удаление логов отменено"; \
	fi

