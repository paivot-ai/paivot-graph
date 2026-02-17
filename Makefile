PLUGIN_DIR := $(shell pwd)
PLUGIN_NAME := paivot-graph

.PHONY: install uninstall test lint help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

install: ## Install plugin locally (claude plugin add)
	claude plugin add "$(PLUGIN_DIR)"
	@echo "$(PLUGIN_NAME) installed. Restart Claude Code sessions for hooks to take effect."

uninstall: ## Remove plugin
	claude plugin remove "$(PLUGIN_NAME)"
	@echo "$(PLUGIN_NAME) removed."

lint: ## Run shellcheck on hook scripts
	shellcheck hooks/*.sh

test: lint ## Run all checks (shellcheck + functional)
	@echo "--- Functional checks ---"
	@echo "Checking hook scripts are executable..."
	@test -x hooks/vault-session-start.sh || (echo "FAIL: vault-session-start.sh not executable" && exit 1)
	@test -x hooks/vault-pre-compact.sh || (echo "FAIL: vault-pre-compact.sh not executable" && exit 1)
	@echo "OK: Hook scripts are executable"
	@echo ""
	@echo "Checking hooks.json is valid JSON..."
	@python3 -c "import json; json.load(open('hooks/hooks.json'))" || (echo "FAIL: hooks.json is not valid JSON" && exit 1)
	@echo "OK: hooks.json is valid JSON"
	@echo ""
	@echo "Checking plugin.json is valid JSON..."
	@python3 -c "import json; json.load(open('.claude-plugin/plugin.json'))" || (echo "FAIL: plugin.json is not valid JSON" && exit 1)
	@echo "OK: plugin.json is valid JSON"
	@echo ""
	@echo "Checking session-start hook exits 0 without obsidian..."
	@echo '{}' | PATH=/usr/bin:/bin hooks/vault-session-start.sh >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: session-start graceful degradation" || echo "FAIL: session-start did not exit 0"
	@echo ""
	@echo "Checking pre-compact hook exits 0..."
	@hooks/vault-pre-compact.sh >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: pre-compact exits 0" || echo "FAIL: pre-compact did not exit 0"
	@echo ""
	@echo "All checks passed."
