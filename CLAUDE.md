# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A **sing-box one-click installation & management script** (by cosaria) for Linux servers. Automates installing, configuring, and managing the [sing-box](https://github.com/SagerNet/sing-box) proxy core with multiple protocol support. Written entirely in Bash.

**Primary language:** Chinese (all user-facing strings, comments, and docs are in Chinese).

## Architecture

### Startup Chain

```
sing-box.sh  →  src/runtime/bootstrap.sh  →  src/runtime/init.sh
     │                    │
     │           bootstrap_main():
     │             1. init_env()           ← from init.sh
     │             2. load_runtime_modules()
     │             3. app_main()           → menu_main()
     │
     sets SCRIPT_VERSION, SH_DIR
```

- `sing-box.sh` — Entry point (symlinked to `/usr/local/bin/sing-box`). Sets `SCRIPT_VERSION` and `SH_DIR`, sources bootstrap.
- `src/runtime/bootstrap.sh` — Orchestrates startup: initializes env, loads all modules in order, calls `app_main()`. Module load order matters (e.g., `protocols/registry.sh` triggers `protocol_load_all` immediately after loading).
- `src/runtime/init.sh` — Pure initialization: paths, platform detection, core status. **Must be passive** (sourcing it must not trigger any action). Defines the `load()` function used to source all other modules.
- `install.sh` — Standalone installer. Downloads core binary, scripts, and jq. Sources `src/core/systemd.sh` directly for service setup.

### Module Layout (`src/`)

| Directory | Purpose |
|---|---|
| `runtime/` | Bootstrap and initialization (no business logic) |
| `ui/` | Terminal UI primitives (`ui.sh`) and interactive menus (`menu.sh`) |
| `protocols/` | Protocol registry + per-protocol implementations |
| `config/` | Config file CRUD (`config.sh`) and share/QR display (`share.sh`) |
| `core/` | Service management (`service.sh`), downloads (`download.sh`), systemd/openrc units (`systemd.sh`) |

### Protocol System

Protocols are self-contained files in `src/protocols/`. Each file must:
1. Set `PROTOCOL_NAME` (e.g., `"Shadowsocks"`)
2. Define `PROTOCOL_EDITABLE` array (fields the user can modify)
3. Implement: `protocol_ask()`, `protocol_json()`, `protocol_url()`, `protocol_info()`, `protocol_parse()`

`src/protocols/registry.sh` provides the registry. `protocol_load_all()` scans `src/protocols/*.sh` (excluding itself), sources each file, and registers it by `PROTOCOL_NAME`. Currently only Shadowsocks is implemented.

### State Management

- `cfg` — Bash associative array (`declare -A cfg`), holds the current config being viewed/edited (keys: `protocol`, `port`, `password`, `method`, `config_file`).
- `CORE_DIR`, `CORE_BIN`, `CONF_DIR`, `CONFIG_JSON`, `LOG_DIR`, `SH_BIN` — Uppercase path globals set by `init_paths()`.
- `ARCH`, `INIT_SYSTEM`, `PKG_CMD` — Platform globals set by `init_platform()`.
- `CORE_VERSION`, `CORE_RUNNING` — Runtime state refreshed by `refresh_core_version()` / `refresh_core_status()`.

### Config Generation Pattern

Configs are per-protocol JSON files stored in `$CONF_DIR/` (e.g., `Shadowsocks-12345.json`). Each contains an `inbounds` array. The main `config.json` holds log/dns/outbound settings. sing-box loads both via `run -c $CONFIG_JSON -C $CONF_DIR`.

## Development

### Testing Locally

```bash
bash install.sh -l     # Install from current directory instead of downloading from GitHub
```

### Running Tests

```bash
bash tests/entrypoint_smoke.sh     # Verifies bootstrap chain, module loading, and menu rendering
bash tests/init_module_smoke.sh    # Verifies init.sh is passive and exposes expected functions
```

Tests use mock binaries and function overrides to test in isolation. No test framework — each script exits 0 on pass, non-zero on fail.

### Install Flags

```bash
bash install.sh -f /path/to/sing-box.tar.gz  # Use custom core binary
bash install.sh -p http://127.0.0.1:2333      # Use proxy for downloads
bash install.sh -v v1.8.13                     # Specify core version
```

### After Installation

`sing-box` command runs the management script, not the raw binary. Use `sing-box bin ...` to invoke the actual sing-box binary.

## Conventions

- **Globals:** UPPER_CASE for all globals (paths, platform, state). No `is_` prefix in the new modules.
- **Module loading:** `load("path/relative/to/src")` sources a module. Modules must be passive (sourcing = define functions only, no side effects).
- **UI functions:** All user-facing output goes through `ui_msg()`, `ui_err()`, `ui_warn()`, `ui_header()`, `ui_menu()`, `ui_ask()` from `src/ui/ui.sh`. Color helpers: `red()`, `green()`, `yellow()`, `cyan()`.
- **Service abstraction:** `_service_do(action)` wraps systemd/openrc differences. Never call `systemctl` or `rc-service` directly in business logic.
- **Error handling:** `ui_err()` prints and exits. Modules return non-zero on failure; `bootstrap_main()` checks each step and aborts on failure.
- **Protocol contract:** Adding a new protocol = one new file in `src/protocols/` implementing the 5 required functions. No registration code needed elsewhere.

## Skill routing

When the user's request matches an available skill, ALWAYS invoke it using the Skill
tool as your FIRST action. Do NOT answer directly, do NOT use other tools first.
The skill has specialized workflows that produce better results than ad-hoc answers.

Key routing rules:
- Product ideas, "is this worth building", brainstorming → invoke office-hours
- Bugs, errors, "why is this broken", 500 errors → invoke investigate
- Ship, deploy, push, create PR → invoke ship
- QA, test the site, find bugs → invoke qa
- Code review, check my diff → invoke review
- Update docs after shipping → invoke document-release
- Weekly retro → invoke retro
- Design system, brand → invoke design-consultation
- Visual audit, design polish → invoke design-review
- Architecture review → invoke plan-eng-review
- Save progress, checkpoint, resume → invoke checkpoint
- Code quality, health check → invoke health
