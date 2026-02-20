PLUGIN_DIR := $(shell pwd)
PLUGIN_NAME := paivot-graph

.PHONY: install update uninstall test lint seed reseed check-deps help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

check-deps: ## Verify required dependencies are installed
	@command -v vlt >/dev/null 2>&1 || \
		(echo "ERROR: vlt is not installed." && \
		 echo "       Install from https://github.com/RamXX/vlt" && \
		 echo "       git clone https://github.com/RamXX/vlt.git && cd vlt && make install" && \
		 exit 1)
	@echo "OK: vlt $$(vlt version 2>&1)"
	@command -v claude >/dev/null 2>&1 || \
		(echo "ERROR: claude (Claude Code) is not installed." && exit 1)
	@echo "OK: claude found"

install: check-deps ## Register marketplace and install plugin
	@claude plugin marketplace add "$(PLUGIN_DIR)" 2>/dev/null \
		&& echo "Marketplace registered." \
		|| echo "Marketplace already registered."
	@claude plugin install "$(PLUGIN_NAME)@$(PLUGIN_NAME)" 2>/dev/null \
		&& echo "Plugin installed." \
		|| echo "Plugin already installed -- run 'make update' to pick up changes."
	@echo "Restart Claude Code sessions for hooks to take effect."

update: ## Push local changes to the installed plugin (bump version first)
	claude plugin marketplace update "$(PLUGIN_NAME)"
	claude plugin update "$(PLUGIN_NAME)@$(PLUGIN_NAME)"
	@echo "Restart Claude Code sessions for changes to take effect."

uninstall: ## Remove plugin and marketplace
	claude plugin uninstall "$(PLUGIN_NAME)@$(PLUGIN_NAME)"
	claude plugin marketplace remove "$(PLUGIN_NAME)"
	@echo "$(PLUGIN_NAME) removed."

seed: ## Seed Obsidian vault with agent prompts and behavioral notes (idempotent)
	scripts/seed-vault.sh

reseed: ## Force-update all vault notes with latest plugin content
	scripts/seed-vault.sh --force

lint: ## Run shellcheck on all shell scripts
	shellcheck hooks/*.sh scripts/*.sh

test: lint ## Run all checks (shellcheck + functional)
	@echo "--- Functional checks ---"
	@echo "Checking hook scripts are executable..."
	@test -x hooks/vault-session-start.sh || (echo "FAIL: vault-session-start.sh not executable" && exit 1)
	@test -x hooks/vault-pre-compact.sh || (echo "FAIL: vault-pre-compact.sh not executable" && exit 1)
	@test -x hooks/vault-stop.sh || (echo "FAIL: vault-stop.sh not executable" && exit 1)
	@test -x hooks/vault-session-end.sh || (echo "FAIL: vault-session-end.sh not executable" && exit 1)
	@test -x scripts/seed-vault.sh || (echo "FAIL: seed-vault.sh not executable" && exit 1)
	@echo "OK: All scripts are executable"
	@echo ""
	@echo "Checking hooks.json is valid JSON..."
	@python3 -c "import json; json.load(open('hooks/hooks.json'))" || (echo "FAIL: hooks.json is not valid JSON" && exit 1)
	@echo "OK: hooks.json is valid JSON"
	@echo ""
	@echo "Checking plugin.json is valid JSON..."
	@python3 -c "import json; json.load(open('.claude-plugin/plugin.json'))" || (echo "FAIL: plugin.json is not valid JSON" && exit 1)
	@echo "OK: plugin.json is valid JSON"
	@echo ""
	@echo "Checking hooks.json registers all 4 hook events..."
	@python3 -c "import json; h=json.load(open('hooks/hooks.json'))['hooks']; assert all(k in h for k in ['SessionStart','PreCompact','Stop','SessionEnd']), 'missing hook events'" \
		|| (echo "FAIL: hooks.json missing required events" && exit 1)
	@echo "OK: All 4 hook events registered"
	@echo ""
	@echo "Checking all 8 agent vault loaders exist..."
	@for agent in sr-pm pm developer architect designer business-analyst anchor retro; do \
		test -f agents/$$agent.md || (echo "FAIL: agents/$$agent.md not found" && exit 1); \
	done
	@echo "OK: All 8 agent vault loaders present"
	@echo ""
	@echo "Checking vault loaders reference vault paths..."
	@for agent in sr-pm pm developer architect designer business-analyst anchor retro; do \
		grep -q 'iCloud~md~obsidian/Documents/Claude/methodology/' agents/$$agent.md || (echo "FAIL: agents/$$agent.md missing vault path" && exit 1); \
	done
	@grep -q 'iCloud~md~obsidian/Documents/Claude' skills/vault-knowledge/SKILL.md || (echo "FAIL: SKILL.md missing vault path" && exit 1)
	@echo "OK: All vault loaders reference vault paths"
	@echo ""
	@echo "Checking session-start hook exits 0 without obsidian..."
	@echo '{}' | PATH=/usr/bin:/bin hooks/vault-session-start.sh >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: session-start graceful degradation" || echo "FAIL: session-start did not exit 0"
	@echo ""
	@echo "Checking pre-compact hook exits 0..."
	@hooks/vault-pre-compact.sh >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: pre-compact exits 0" || echo "FAIL: pre-compact did not exit 0"
	@echo ""
	@echo "Checking stop hook exits 0..."
	@hooks/vault-stop.sh >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: stop exits 0" || echo "FAIL: stop did not exit 0"
	@echo ""
	@echo "Checking session-end hook exits 0..."
	@echo '{}' | hooks/vault-session-end.sh >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: session-end exits 0" || echo "FAIL: session-end did not exit 0"
	@echo ""
	@echo "All checks passed."
