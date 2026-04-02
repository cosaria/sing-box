# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**sing-box panel** — a TUI-based management panel for [sing-box](https://github.com/SagerNet/sing-box) proxy core. Single Go binary embedding sing-box as a library, with SQLite storage, multi-protocol support, and terminal UI for interactive management.

**Primary language:** Go. All user-facing strings in Chinese.

## Architecture

```
sing-box serve (systemd daemon)          sing-box (TUI management)
├── Engine (sing-box core)                ├── Direct SQLite access
├── Stats Collector (60s interval)        ├── Add/Edit/Delete inbounds
├── PID file (for SIGHUP signal)          ├── View share links + QR codes
└── SIGHUP → reload engine config         └── SIGHUP → notify daemon to reload
```

**No HTTP API.** TUI talks directly to SQLite. Daemon and TUI coordinate via PID file + Unix signals.

### Module Layout (`internal/`)

| Directory | Purpose |
|---|---|
| `store/` | SQLite connection, migrations, inbound CRUD, traffic log queries |
| `engine/` | sing-box lifecycle (Start/Stop/Reload), config builder, crash recovery |
| `protocol/` | Protocol interface + registry, Shadowsocks/VLESS/Trojan implementations |
| `stats/` | ConnectionTracker for per-inbound traffic counting, periodic DB flush |
| `tui/` | Bubbletea TUI: menu, add wizard, list/detail, edit, stats, status, QR |
| `platform/` | OS/arch/init-system detection, path constants |
| `service/` | systemd/OpenRC service install/uninstall with path validation |
| `updater/` | Self-update via GitHub Releases with atomic binary replace |

### Protocol System

Protocols are self-contained files in `internal/protocol/`. Each implements the `Protocol` interface:
- `Name()`, `DisplayName()` — identification
- `DefaultSettings(port)` — auto-generate config (UUID passwords, X25519 keys)
- `BuildInbound(ib)` — convert store.Inbound to sing-box option.Inbound
- `GenerateURL(ib, host)` — produce share link (ss://, vless://, trojan://)

Protocols self-register via `init()`. Registry: `protocol.Get("shadowsocks")`.

Currently implemented: **Shadowsocks (SS2022)**, **VLESS-REALITY**, **Trojan**.

### State Management

- `store.Store` — SQLite with WAL mode, busy_timeout(5000), migrations
- `store.Inbound` — tag, protocol, port, settings (JSON string)
- `store.TrafficLog` / `TrafficSummary` — per-inbound upload/download stats
- `engine.Engine` — mutex-protected sing-box box.Box lifecycle, crash counter (3 in 60s = refuse)
- `stats.Tracker` — atomic per-inbound byte counters via ConnectionTracker interface

### Cobra Commands

```
sing-box              # TUI mode (interactive management)
sing-box serve        # Daemon mode (engine + stats, for systemd)
sing-box update       # Self-update from GitHub Releases
sing-box service install/uninstall  # System service management
sing-box version      # Print version info
```

## Development

### Running Tests

```bash
go test ./... -v -count=1 -race
```

Tests use in-memory SQLite (`:memory:`). No external dependencies needed.

### Building

```bash
go build -o sing-box ./cmd/sing-box/
```

Cross-compile:
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o sing-box-linux-amd64 ./cmd/sing-box/
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o sing-box-linux-arm64 ./cmd/sing-box/
```

Version injection:
```bash
go build -ldflags "-X main.Version=v0.1.0" -o sing-box ./cmd/sing-box/
```

### Install (local dev)

```bash
bash install.sh -l    # Build and install from current directory
```

## Conventions

- **Globals:** UPPER_CASE for constants, CamelCase for exported Go symbols
- **Module path:** `github.com/cosaria/sing-box`
- **Protocol contract:** Adding a new protocol = one new file in `internal/protocol/` implementing the Protocol interface with `init()` self-registration
- **Service management:** Never call systemctl/rc-service directly — use `internal/service/` abstraction
- **TUI pattern:** Bubbletea Model/Update/View, state machine in tui.go, one file per view
- **Error handling:** Wrap with context (`fmt.Errorf("failed to X: %w", err)`), engine panics caught by recover
- **Path validation:** Service install paths validated against `^[a-zA-Z0-9/._-]+$`

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
