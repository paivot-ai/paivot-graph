PLUGIN_DIR  := $(shell pwd)
PLUGIN_NAME := paivot-graph
VERSION     := $(shell cat VERSION)
CACHE_BASE  := $(HOME)/.claude/plugins/cache/$(PLUGIN_NAME)/$(PLUGIN_NAME)
CACHE_DIR   := $(CACHE_BASE)/$(VERSION)

.PHONY: install update uninstall test lint seed reseed check-deps \
        fetch-vlt-skill update-vlt-skill help build-pvg test-pvg \
        sync-cache

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'

# ---------------------------------------------------------------------------
# Dependency checks
# ---------------------------------------------------------------------------

check-deps: ## Verify required dependencies are installed
	@command -v go >/dev/null 2>&1 || \
		(echo "ERROR: go is not installed (required to build pvg)." && \
		 echo "       Install from https://go.dev/dl/" && exit 1)
	@echo "OK: go $$(go version | awk '{print $$3}')"
	@command -v vlt >/dev/null 2>&1 || \
		(echo "ERROR: vlt is not installed." && \
		 echo "       Install from https://github.com/RamXX/vlt" && \
		 echo "       git clone https://github.com/RamXX/vlt.git && cd vlt && make install" && \
		 exit 1)
	@echo "OK: vlt $$(vlt version 2>&1)"
	@command -v claude >/dev/null 2>&1 || \
		(echo "ERROR: claude (Claude Code) is not installed." && exit 1)
	@echo "OK: claude found"
	@command -v nd >/dev/null 2>&1 && \
		echo "OK: nd found at $$(command -v nd)" || \
		echo "WARN: nd not installed (needed for execution agents -- install from https://github.com/RamXX/nd)"

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------

build-pvg: ## Build the pvg Go CLI (fetches Go modules if needed)
	cd pvg-cli && go mod download && \
		go build -ldflags "-X main.version=$(VERSION)" -o ../bin/pvg ./cmd/pvg/
	@echo "Built bin/pvg $(VERSION)"

test-pvg: ## Run pvg Go tests
	cd pvg-cli && go test ./... -v

# ---------------------------------------------------------------------------
# Plugin cache sync -- copy pvg binary into the Claude Code plugin cache
# ---------------------------------------------------------------------------

sync-cache: build-pvg ## Copy pvg binary to plugin cache so hooks work at runtime
	@if [ -d "$(CACHE_DIR)" ]; then \
		mkdir -p "$(CACHE_DIR)/bin" && \
		cp bin/pvg "$(CACHE_DIR)/bin/pvg" && \
		echo "OK: pvg synced to plugin cache ($(CACHE_DIR)/bin/pvg)"; \
	else \
		echo "WARN: Plugin cache dir not found at $(CACHE_DIR)"; \
		echo "      Run 'make install' first, then 'make sync-cache'."; \
	fi

# ---------------------------------------------------------------------------
# Install / update / uninstall
# ---------------------------------------------------------------------------

install: check-deps build-pvg fetch-vlt-skill ## Full install: deps, build, marketplace, plugin, cache sync
	@claude plugin marketplace add "$(PLUGIN_DIR)" 2>/dev/null \
		&& echo "Marketplace registered." \
		|| echo "Marketplace already registered."
	@claude plugin install "$(PLUGIN_NAME)@$(PLUGIN_NAME)" 2>/dev/null \
		&& echo "Plugin installed." \
		|| echo "Plugin already installed -- run 'make update' to pick up changes."
	@$(MAKE) --no-print-directory sync-cache
	@echo ""
	@echo "Install complete (v$(VERSION)). Restart Claude Code sessions for hooks to take effect."

update: check-deps build-pvg update-vlt-skill ## Update plugin, vlt skill, and sync binary to cache
	claude plugin marketplace update "$(PLUGIN_NAME)"
	claude plugin update "$(PLUGIN_NAME)@$(PLUGIN_NAME)"
	@$(MAKE) --no-print-directory sync-cache
	@echo ""
	@echo "Update complete (v$(VERSION)). Restart Claude Code sessions for changes to take effect."

uninstall: ## Remove plugin and marketplace
	claude plugin uninstall "$(PLUGIN_NAME)@$(PLUGIN_NAME)"
	claude plugin marketplace remove "$(PLUGIN_NAME)"
	@echo "$(PLUGIN_NAME) removed."

# ---------------------------------------------------------------------------
# Vault seeding
# ---------------------------------------------------------------------------

seed: build-pvg ## Seed Obsidian vault with agent prompts and behavioral notes (idempotent)
	bin/pvg seed

reseed: build-pvg ## Force-update all vault notes with latest plugin content
	bin/pvg seed --force

# ---------------------------------------------------------------------------
# vlt skill management
# ---------------------------------------------------------------------------

fetch-vlt-skill: ## Fetch and install the vlt skill from GitHub (skips if present)
	scripts/fetch-vlt-skill.sh

update-vlt-skill: ## Force-update the vlt skill from GitHub
	scripts/fetch-vlt-skill.sh --force

# ---------------------------------------------------------------------------
# Lint & test
# ---------------------------------------------------------------------------

lint: ## Run shellcheck on shell scripts
	shellcheck scripts/fetch-vlt-skill.sh

test: lint test-pvg build-pvg ## Run all checks (shellcheck + Go tests + functional)
	@echo "--- Functional checks ---"
	@echo "Checking pvg binary exists..."
	@test -x bin/pvg || (echo "FAIL: bin/pvg not found or not executable" && exit 1)
	@echo "OK: bin/pvg built"
	@echo ""
	@echo "Checking scripts are executable..."
	@test -x scripts/fetch-vlt-skill.sh || (echo "FAIL: fetch-vlt-skill.sh not executable" && exit 1)
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
	@echo "Checking version sync (VERSION, plugin.json, marketplace.json)..."
	@python3 -c "\
v_file = open('VERSION').read().strip(); \
import json; \
v_plugin = json.load(open('.claude-plugin/plugin.json'))['version']; \
v_market = json.load(open('.claude-plugin/marketplace.json'))['plugins'][0]['version']; \
assert v_file == v_plugin == v_market, \
    f'Version mismatch: VERSION={v_file}, plugin.json={v_plugin}, marketplace.json={v_market}'" \
		|| (echo "FAIL: version mismatch across VERSION, plugin.json, marketplace.json" && exit 1)
	@echo "OK: All versions in sync ($(VERSION))"
	@echo ""
	@echo "Checking hooks.json registers all 5 hook events..."
	@python3 -c "import json; h=json.load(open('hooks/hooks.json'))['hooks']; assert all(k in h for k in ['PreToolUse','SessionStart','PreCompact','Stop','SessionEnd']), 'missing hook events'" \
		|| (echo "FAIL: hooks.json missing required events" && exit 1)
	@echo "OK: All 5 hook events registered"
	@echo ""
	@echo "Checking hooks.json points to pvg binary..."
	@python3 -c "import json; h=json.load(open('hooks/hooks.json'))['hooks']; cmds=[hook.get('command','') for e in h.values() for entry in e for hook in entry.get('hooks',[])]; bad=[c for c in cmds if 'bin/pvg' not in c]; assert not bad, f'hooks not using pvg: {bad}'" \
		|| (echo "FAIL: hooks.json has hooks not using pvg" && exit 1)
	@echo "OK: All hooks use pvg binary"
	@echo ""
	@echo "Checking all 8 agent vault loaders exist..."
	@for agent in sr-pm pm developer architect designer business-analyst anchor retro; do \
		test -f agents/$$agent.md || (echo "FAIL: agents/$$agent.md not found" && exit 1); \
	done
	@echo "OK: All 8 agent vault loaders present"
	@echo ""
	@echo "Checking vault loaders use vlt commands (not hardcoded paths)..."
	@for agent in sr-pm pm developer architect designer business-analyst anchor retro; do \
		grep -q 'vlt vault="Claude" read file=' agents/$$agent.md || (echo "FAIL: agents/$$agent.md missing vlt read command" && exit 1); \
	done
	@grep -q 'vlt vault="Claude" read file=' skills/vault-knowledge/SKILL.md || (echo "FAIL: SKILL.md missing vlt read command" && exit 1)
	@echo "OK: All vault loaders use dynamic vlt commands"
	@echo ""
	@echo "Checking pvg guard allows non-vault Edit..."
	@echo '{"tool_name":"Edit","tool_input":{"file_path":"/tmp/safe.md"}}' | bin/pvg guard >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: pvg guard allows non-vault Edit" || echo "FAIL: pvg guard blocked a safe Edit"
	@echo ""
	@echo "Checking pvg guard blocks vault methodology/ Edit..."
	@echo '{"tool_name":"Edit","tool_input":{"file_path":"$(HOME)/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude/methodology/Developer Agent.md"}}' \
		| bin/pvg guard >/dev/null 2>&1; \
		test $$? -eq 2 && echo "OK: pvg guard blocks methodology/ Edit" || echo "FAIL: pvg guard did not block methodology/ Edit"
	@echo ""
	@echo "Checking pvg guard blocks vault conventions/ Write..."
	@echo '{"tool_name":"Write","tool_input":{"file_path":"$(HOME)/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude/conventions/Session Operating Mode.md"}}' \
		| bin/pvg guard >/dev/null 2>&1; \
		test $$? -eq 2 && echo "OK: pvg guard blocks conventions/ Write" || echo "FAIL: pvg guard did not block conventions/ Write"
	@echo ""
	@echo "Checking pvg guard blocks vault decisions/ Edit..."
	@echo '{"tool_name":"Edit","tool_input":{"file_path":"$(HOME)/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude/decisions/Some Decision.md"}}' \
		| bin/pvg guard >/dev/null 2>&1; \
		test $$? -eq 2 && echo "OK: pvg guard blocks decisions/ Edit" || echo "FAIL: pvg guard did not block decisions/ Edit"
	@echo ""
	@echo "Checking pvg guard allows vault _inbox/ Write..."
	@echo '{"tool_name":"Write","tool_input":{"file_path":"$(HOME)/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude/_inbox/Proposal.md"}}' \
		| bin/pvg guard >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: pvg guard allows _inbox/ Write" || echo "FAIL: pvg guard blocked _inbox/ Write"
	@echo ""
	@echo "Checking pvg guard allows safe Bash commands..."
	@echo '{"tool_name":"Bash","tool_input":{"command":"ls /tmp"}}' | bin/pvg guard >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: pvg guard allows safe Bash" || echo "FAIL: pvg guard blocked safe Bash"
	@echo ""
	@echo "Checking pvg guard allows vlt Bash commands..."
	@echo '{"tool_name":"Bash","tool_input":{"command":"vlt vault=\"Claude\" append file=\"Sr PM Agent\" content=\"test\""}}' \
		| bin/pvg guard >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: pvg guard allows vlt commands" || echo "FAIL: pvg guard blocked vlt"
	@echo ""
	@echo "Checking pvg version..."
	@bin/pvg version | grep -q "$(VERSION)" && echo "OK: pvg version matches VERSION file" || echo "FAIL: pvg version mismatch"
	@echo ""
	@echo "Checking pvg hook session-start exits 0..."
	@echo '{}' | bin/pvg hook session-start >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: pvg hook session-start exits 0" || echo "FAIL: pvg hook session-start did not exit 0"
	@echo ""
	@echo "Checking pvg hook pre-compact exits 0..."
	@bin/pvg hook pre-compact >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: pvg hook pre-compact exits 0" || echo "FAIL: pvg hook pre-compact did not exit 0"
	@echo ""
	@echo "Checking pvg hook stop exits 0..."
	@bin/pvg hook stop >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: pvg hook stop exits 0" || echo "FAIL: pvg hook stop did not exit 0"
	@echo ""
	@echo "Checking pvg hook session-end exits 0..."
	@echo '{}' | bin/pvg hook session-end >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: pvg hook session-end exits 0" || echo "FAIL: pvg hook session-end did not exit 0"
	@echo ""
	@echo "Checking pvg binary in plugin cache..."
	@if [ -d "$(CACHE_DIR)" ]; then \
		test -x "$(CACHE_DIR)/bin/pvg" \
			&& echo "OK: pvg exists in plugin cache" \
			|| echo "WARN: pvg not in plugin cache (run 'make sync-cache')"; \
	else \
		echo "SKIP: plugin cache not found (plugin not installed yet)"; \
	fi
	@echo ""
	@echo "All checks passed."
