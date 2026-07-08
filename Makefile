.PHONY: build run dev frontend service restart uninstall clean help

help: ## show this help
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | sed 's/:.*##/\t/' | sort

build: ## build frontend + single Go binary
	@./build.sh

run: build ## build then run (auto-opens browser)
	@./claude-deck

dev: ## frontend hot-reload dev server (proxies /api -> :7420)
	@cd web/frontend && npm install && npm run dev

frontend: ## build only the frontend into web/static
	@cd web/frontend && npm install && npm run build

service: build ## install as an always-on macOS launchd agent
	@./deploy/install-service.sh

restart: build ## rebuild + restart the running launchd agent (apply code changes)
	@launchctl kickstart -k gui/$$(id -u)/com.claudedeck.agent && echo "✓ rebuilt and restarted the claude-deck agent"

uninstall: ## remove the launchd agent
	@./deploy/uninstall-service.sh

clean: ## remove build artifacts
	@rm -f claude-deck && rm -rf web/static/assets web/frontend/node_modules
