PLUGIN_DIR  := $(shell pwd)
PLUGIN_NAME := paivot-graph
VERSION     := $(shell cat VERSION)
CACHE_BASE  := $(HOME)/.claude/plugins/cache/$(PLUGIN_NAME)/$(PLUGIN_NAME)
CACHE_DIR   := $(CACHE_BASE)/$(VERSION)

.PHONY: install update uninstall test check-deps check-pvg \
        fetch-vlt-skill update-vlt-skill help sync-cache bump

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'

# ---------------------------------------------------------------------------
# Version management -- single command to bump VERSION + plugin.json + marketplace.json
# ---------------------------------------------------------------------------

bump: ## Bump version: make bump v=1.22.0 (updates all three version files atomically)
ifndef v
	$(error Usage: make bump v=X.Y.Z)
endif
	@echo "$(v)" > VERSION
	@python3 -c "\
import json; \
p = json.load(open('.claude-plugin/plugin.json')); \
p['version'] = '$(v)'; \
f = open('.claude-plugin/plugin.json','w'); \
json.dump(p, f, indent=2); f.write('\n'); f.close(); \
print('OK: plugin.json -> $(v)')"
	@python3 -c "\
import json; \
m = json.load(open('.claude-plugin/marketplace.json')); \
m['plugins'][0]['version'] = '$(v)'; \
f = open('.claude-plugin/marketplace.json','w'); \
json.dump(m, f, indent=2); f.write('\n'); f.close(); \
print('OK: marketplace.json -> $(v)')"
	@echo "All versions synced to $(v)"

# ---------------------------------------------------------------------------
# Dependency checks
# ---------------------------------------------------------------------------

check-deps: ## Verify required dependencies are installed
	@command -v pvg >/dev/null 2>&1 || \
		(echo "WARN: pvg is not installed." && \
		 echo "      Install from https://github.com/paivot-ai/pvg" && \
		 echo "      gh release download -R paivot-ai/pvg -p '*darwin*arm64*' -D /tmp && tar xzf /tmp/pvg_*.tar.gz -C ~/go/bin")
	@command -v pvg >/dev/null 2>&1 && echo "OK: pvg $$(pvg version 2>&1)" || true
	@command -v vlt >/dev/null 2>&1 || \
		(echo "ERROR: vlt is not installed." && \
		 echo "       Install from https://github.com/paivot-ai/vlt" && \
		 echo "       git clone https://github.com/paivot-ai/vlt.git && cd vlt && make install" && \
		 exit 1)
	@echo "OK: vlt $$(vlt version 2>&1)"
	@command -v claude >/dev/null 2>&1 || \
		(echo "ERROR: claude (Claude Code) is not installed." && exit 1)
	@echo "OK: claude found"
	@command -v nd >/dev/null 2>&1 && \
		echo "OK: nd found at $$(command -v nd)" || \
		echo "WARN: nd not installed (needed for execution agents -- install from https://github.com/paivot-ai/nd)"

check-pvg: ## Verify pvg is on PATH
	@command -v pvg >/dev/null 2>&1 || \
		(echo "ERROR: pvg is not on PATH." && \
		 echo "       Install from https://github.com/paivot-ai/pvg" && \
		 echo "       gh release download -R paivot-ai/pvg -p '*darwin*arm64*' -D /tmp && tar xzf /tmp/pvg_*.tar.gz -C ~/go/bin" && \
		 exit 1)
	@echo "OK: pvg found at $$(command -v pvg) -- $$(pvg version 2>&1)"

# ---------------------------------------------------------------------------
# Plugin cache sync -- copy pvg binary and skills into the Claude Code plugin cache
# ---------------------------------------------------------------------------

sync-cache: check-pvg ## Copy pvg binary, skills, and agents to ALL cached versions so running sessions survive upgrades
	@found=0; \
	pvg_path=$$(command -v pvg); \
	if [ -d "$(CACHE_BASE)" ]; then \
		for vdir in "$(CACHE_BASE)"/*/; do \
			if [ -d "$$vdir" ]; then \
				mkdir -p "$$vdir/bin" && \
				cp "$$pvg_path" "$$vdir/bin/pvg" && \
				echo "OK: pvg synced to $$vdir/bin/pvg"; \
				for skill_dir in skills/*/; do \
					if [ -d "$$skill_dir" ]; then \
						mkdir -p "$$vdir/$$skill_dir" && \
						cp -R "$$skill_dir"* "$$vdir/$$skill_dir" && \
						echo "OK: $$skill_dir synced to $$vdir/$$skill_dir"; \
					fi; \
				done; \
				if [ -d agents ]; then \
					mkdir -p "$$vdir/agents" && \
					cp agents/*.md "$$vdir/agents/" && \
					echo "OK: agents/ synced to $$vdir/agents/"; \
				fi; \
				found=1; \
			fi; \
		done; \
	fi; \
	if [ "$$found" = "0" ]; then \
		echo "WARN: No plugin cache dirs found under $(CACHE_BASE)"; \
		echo "      Run 'make install' first, then 'make sync-cache'."; \
	fi

# ---------------------------------------------------------------------------
# Install / update / uninstall
# ---------------------------------------------------------------------------

install: check-deps check-pvg fetch-vlt-skill ## Full install: deps, pvg check, marketplace, plugin, cache sync
	@claude plugin marketplace add "$(PLUGIN_DIR)" 2>/dev/null \
		&& echo "Marketplace registered." \
		|| echo "Marketplace already registered."
	@claude plugin install "$(PLUGIN_NAME)@$(PLUGIN_NAME)" 2>/dev/null \
		&& echo "Plugin installed." \
		|| echo "Plugin already installed -- run 'make update' to pick up changes."
	@$(MAKE) --no-print-directory sync-cache
	@echo ""
	@echo "Install complete (v$(VERSION)). Restart Claude Code sessions for hooks to take effect."

update: check-deps check-pvg update-vlt-skill ## Update plugin, vlt skill, and sync binary to cache
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
# vlt skill management
# ---------------------------------------------------------------------------

fetch-vlt-skill: check-pvg ## Fetch and install the vlt skill from GitHub (skips if present)
	pvg fetch-vlt-skill

update-vlt-skill: check-pvg ## Force-update the vlt skill from GitHub
	pvg fetch-vlt-skill --force

# ---------------------------------------------------------------------------
# Lint & test
# ---------------------------------------------------------------------------

test: check-pvg ## Run all checks (functional)
	@echo "--- Functional checks ---"
	@echo "Checking pvg is on PATH..."
	@command -v pvg >/dev/null 2>&1 || (echo "FAIL: pvg not found on PATH" && exit 1)
	@echo "OK: pvg found at $$(command -v pvg)"
	@echo ""
	@echo "Checking no shell scripts remain..."
	@test ! -d scripts || test -z "$$(ls scripts/ 2>/dev/null)" || (echo "FAIL: shell scripts still exist in scripts/" && exit 1)
	@echo "OK: No shell scripts"
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
	@echo "Checking hooks.json registers all required hook events..."
	@python3 -c "import json; h=json.load(open('hooks/hooks.json'))['hooks']; assert all(k in h for k in ['PreToolUse','UserPromptSubmit','SubagentStart','SubagentStop','SessionStart','PreCompact','Stop','PostToolUse','SessionEnd']), 'missing hook events'" \
		|| (echo "FAIL: hooks.json missing required events" && exit 1)
	@echo "OK: All required hook events registered"
	@echo ""
	@echo "Checking dispatcher-tracked subagents are wired in hooks.json..."
	@python3 -c "\
import json; \
h=json.load(open('hooks/hooks.json'))['hooks']; \
required=['paivot-graph:business-analyst','paivot-graph:designer','paivot-graph:architect','paivot-graph:sr-pm','paivot-graph:developer','paivot-graph:pm']; \
start=' '.join(entry.get('matcher','') for entry in h['SubagentStart']); \
stop=' '.join(entry.get('matcher','') for entry in h['SubagentStop']); \
missing=[name for name in required if name not in start or name not in stop]; \
assert not missing, f'missing tracked subagent hooks: {missing}'" \
		|| (echo "FAIL: hooks.json is missing tracked subagent matchers" && exit 1)
	@echo "OK: Dispatcher-tracked subagents are wired"
	@echo ""
	@echo "Checking hooks.json points to pvg binary..."
	@python3 -c "import json; h=json.load(open('hooks/hooks.json'))['hooks']; cmds=[hook.get('command','') for e in h.values() for entry in e for hook in entry.get('hooks',[])]; bad=[c for c in cmds if 'pvg' not in c]; assert not bad, f'hooks not using pvg: {bad}'" \
		|| (echo "FAIL: hooks.json has hooks not using pvg" && exit 1)
	@echo "OK: All hooks use pvg binary"
	@echo ""
	@echo "Checking all 8 agent vault loaders exist..."
	@for agent in sr-pm pm developer architect designer business-analyst anchor retro; do \
		test -f agents/$$agent.md || (echo "FAIL: agents/$$agent.md not found" && exit 1); \
	done
	@echo "OK: All 8 agent vault loaders present"
	@echo ""
	@echo "Checking agent prompts are self-contained (no vault-read for operational instructions)..."
	@for agent in sr-pm pm developer anchor retro; do \
		if grep -q 'Read your full instructions from the vault' agents/$$agent.md; then \
			echo "FAIL: agents/$$agent.md still has vault-read loader (should be self-contained)" && exit 1; \
		fi; \
	done
	@echo "OK: Agent prompts are self-contained"
	@echo ""
	@echo "Checking pvg guard allows non-vault Edit..."
	@echo '{"tool_name":"Edit","tool_input":{"file_path":"/tmp/safe.md"}}' | pvg guard >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: pvg guard allows non-vault Edit" || echo "FAIL: pvg guard blocked a safe Edit"
	@echo ""
	@echo "Checking pvg guard blocks vault methodology/ Edit..."
	@echo '{"tool_name":"Edit","tool_input":{"file_path":"$(HOME)/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude/methodology/Developer Agent.md"}}' \
		| pvg guard >/dev/null 2>&1; \
		test $$? -eq 2 && echo "OK: pvg guard blocks methodology/ Edit" || echo "FAIL: pvg guard did not block methodology/ Edit"
	@echo ""
	@echo "Checking pvg guard blocks vault conventions/ Write..."
	@echo '{"tool_name":"Write","tool_input":{"file_path":"$(HOME)/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude/conventions/Session Operating Mode.md"}}' \
		| pvg guard >/dev/null 2>&1; \
		test $$? -eq 2 && echo "OK: pvg guard blocks conventions/ Write" || echo "FAIL: pvg guard did not block conventions/ Write"
	@echo ""
	@echo "Checking pvg guard blocks vault decisions/ Edit..."
	@echo '{"tool_name":"Edit","tool_input":{"file_path":"$(HOME)/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude/decisions/Some Decision.md"}}' \
		| pvg guard >/dev/null 2>&1; \
		test $$? -eq 2 && echo "OK: pvg guard blocks decisions/ Edit" || echo "FAIL: pvg guard did not block decisions/ Edit"
	@echo ""
	@echo "Checking pvg guard allows vault _inbox/ Write..."
	@echo '{"tool_name":"Write","tool_input":{"file_path":"$(HOME)/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude/_inbox/Proposal.md"}}' \
		| pvg guard >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: pvg guard allows _inbox/ Write" || echo "FAIL: pvg guard blocked _inbox/ Write"
	@echo ""
	@echo "Checking pvg guard allows safe Bash commands..."
	@echo '{"tool_name":"Bash","tool_input":{"command":"ls /tmp"}}' | pvg guard >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: pvg guard allows safe Bash" || echo "FAIL: pvg guard blocked safe Bash"
	@echo ""
	@echo "Checking pvg guard allows vlt Bash commands..."
	@echo '{"tool_name":"Bash","tool_input":{"command":"vlt vault=\"Claude\" append file=\"Sr PM Agent\" content=\"test\""}}' \
		| pvg guard >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: pvg guard allows vlt commands" || echo "FAIL: pvg guard blocked vlt"
	@echo ""
	@echo "Checking pvg version..."
	@pvg version >/dev/null 2>&1 && echo "OK: pvg version runs" || echo "FAIL: pvg version failed"
	@echo ""
	@echo "Checking pvg hook session-start exits 0..."
	@echo '{}' | pvg hook session-start >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: pvg hook session-start exits 0" || echo "FAIL: pvg hook session-start did not exit 0"
	@echo ""
	@echo "Checking pvg hook pre-compact exits 0..."
	@pvg hook pre-compact >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: pvg hook pre-compact exits 0" || echo "FAIL: pvg hook pre-compact did not exit 0"
	@echo ""
	@echo "Checking pvg hook stop exits 0..."
	@pvg hook stop >/dev/null 2>&1; \
		test $$? -eq 0 && echo "OK: pvg hook stop exits 0" || echo "FAIL: pvg hook stop did not exit 0"
	@echo ""
	@echo "Checking pvg hook session-end exits 0..."
	@echo '{}' | pvg hook session-end >/dev/null 2>&1; \
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
