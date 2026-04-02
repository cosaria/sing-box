# M1: Core Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the core backend for sing-box panel: SQLite store, engine lifecycle, platform detection, service management, and HTTP API with inbound CRUD + engine control.

**Architecture:** API-First single binary. HTTP API (chi/v5) is the only control plane. SQLite stores inbound configs and settings. Engine wraps sing-box `box.New()/Start()/Close()` with reload and crash recovery. Platform layer detects init system for service management.

**Tech Stack:** Go 1.26, sing-box v1.13.5 (embedded), chi/v5, modernc.org/sqlite (pure Go, no CGO), cobra CLI

**Module path:** `github.com/cosaria/sing-box`

---

## File Structure

```
cmd/sing-box/main.go                  - Cobra CLI entry point with `serve` command
internal/
├── store/
│   ├── store.go                      - SQLite connection, migrations, settings KV
│   ├── store_test.go                 - Migration and settings tests
│   ├── inbound.go                    - Inbound model + CRUD
│   └── inbound_test.go              - Inbound CRUD tests
├── engine/
│   ├── engine.go                     - sing-box lifecycle (Start/Stop/Reload)
│   ├── build.go                      - Build sing-box options from store inbounds
│   └── build_test.go                - Build logic tests (no network needed)
├── platform/
│   ├── platform.go                   - OS/arch/init-system detection + paths
│   └── platform_test.go             - Detection tests
├── service/
│   ├── service.go                    - Manager interface + NewManager factory
│   ├── systemd.go                    - systemd implementation
│   ├── openrc.go                     - OpenRC implementation
│   └── service_test.go              - Template generation tests
└── api/
    ├── server.go                     - HTTP server + chi router setup
    ├── middleware.go                 - Token auth middleware
    ├── inbound.go                    - Inbound CRUD handlers
    ├── engine.go                     - Engine status + reload handlers
    └── api_test.go                   - HTTP handler tests
```

## Dependency Graph (parallelizable groups)

```
Group A (parallel):     store  ║  platform
Group B (parallel):     engine (needs store)  ║  service (needs platform)
Group C (sequential):   api (needs engine + store + service)
Group D (sequential):   main (needs everything)
```

---

### Task 1: Add Dependencies + Project Scaffold

**Files:**
- Modify: `go.mod`
- Create: all `internal/` directories

- [ ] **Step 1: Add new direct dependencies**

```bash
cd /Users/admin/Codes/ProxyCode/sing-box
go get github.com/spf13/cobra@latest
go get modernc.org/sqlite@latest
go get github.com/go-chi/chi/v5@v5.2.5
go get github.com/go-chi/render@v1.0.3
```

- [ ] **Step 2: Create directory structure**

```bash
mkdir -p internal/store internal/engine internal/platform internal/service internal/api
```

- [ ] **Step 3: Verify go.mod has all direct deps**

Run: `grep -E "cobra|modernc|chi" go.mod`

Expected: All four packages listed as direct (not `// indirect`).

If any still show `// indirect`, manually edit `go.mod` to move them to the direct `require` block.

- [ ] **Step 4: Commit scaffold**

```bash
git add go.mod go.sum internal/
git commit -m "chore: add M1 dependencies and project scaffold"
```

---

### Task 2: SQLite Store with Migrations

**Files:**
- Create: `internal/store/store.go`
- Create: `internal/store/store_test.go`

- [ ] **Step 1: Write store tests**

Create `internal/store/store_test.go`:

```go
package store

import (
	"testing"
)

func TestOpenInMemory(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) failed: %v", err)
	}
	defer s.Close()
}

func TestMigrationsAreIdempotent(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// Open again on same DB should not fail (migrations already applied)
	// We test by calling migrate() again directly
	if err := s.migrate(); err != nil {
		t.Fatalf("second migrate() failed: %v", err)
	}
}

func TestSettings(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// Get non-existent key returns empty string and no error
	val, err := s.GetSetting("nonexistent")
	if err != nil {
		t.Fatalf("GetSetting(nonexistent) error: %v", err)
	}
	if val != "" {
		t.Fatalf("expected empty string, got %q", val)
	}

	// Set and get
	if err := s.SetSetting("api_token", "secret123"); err != nil {
		t.Fatalf("SetSetting error: %v", err)
	}
	val, err = s.GetSetting("api_token")
	if err != nil {
		t.Fatalf("GetSetting error: %v", err)
	}
	if val != "secret123" {
		t.Fatalf("expected 'secret123', got %q", val)
	}

	// Overwrite
	if err := s.SetSetting("api_token", "updated"); err != nil {
		t.Fatalf("SetSetting overwrite error: %v", err)
	}
	val, _ = s.GetSetting("api_token")
	if val != "updated" {
		t.Fatalf("expected 'updated', got %q", val)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/store/ -v -count=1`

Expected: Compilation error — package `store` has no Go files yet.

- [ ] **Step 3: Implement store.go**

Create `internal/store/store.go`:

```go
package store

import (
	"database/sql"
	"fmt"
	"strconv"

	_ "modernc.org/sqlite"
)

// Store wraps a SQLite database for panel state.
type Store struct {
	db *sql.DB
}

// Open creates or opens a SQLite database at dbPath and runs migrations.
// Use ":memory:" for testing.
func Open(dbPath string) (*Store, error) {
	dsn := dbPath
	if dbPath != ":memory:" {
		dsn = fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", dbPath)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite single-writer

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}
	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB exposes the underlying *sql.DB for use by other packages (e.g., transactions).
func (s *Store) DB() *sql.DB {
	return s.db
}

var migrations = []string{
	// Migration 1: inbounds table
	`CREATE TABLE IF NOT EXISTS inbounds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tag TEXT UNIQUE NOT NULL,
		protocol TEXT NOT NULL,
		port INTEGER NOT NULL,
		settings TEXT NOT NULL DEFAULT '{}',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
}

func (s *Store) migrate() error {
	// Bootstrap: ensure settings table exists before anything else
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("failed to create settings table: %w", err)
	}

	// Read current schema version
	var versionStr string
	err := s.db.QueryRow("SELECT value FROM settings WHERE key = 'schema_version'").Scan(&versionStr)
	version := 0
	if err == nil {
		version, _ = strconv.Atoi(versionStr)
	}

	// Apply pending migrations
	for i := version; i < len(migrations); i++ {
		if _, err := s.db.Exec(migrations[i]); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	// Update schema version
	_, err = s.db.Exec(
		"INSERT OR REPLACE INTO settings (key, value) VALUES ('schema_version', ?)",
		strconv.Itoa(len(migrations)),
	)
	return err
}

// GetSetting returns the value for a key, or empty string if not found.
func (s *Store) GetSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetSetting upserts a key-value pair in the settings table.
func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)",
		key, value,
	)
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/store/ -v -count=1 -race`

Expected: All 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/store/store.go internal/store/store_test.go
git commit -m "feat(store): SQLite store with migrations and settings KV"
```

---

### Task 3: Inbound CRUD Operations

**Files:**
- Create: `internal/store/inbound.go`
- Create: `internal/store/inbound_test.go`

- [ ] **Step 1: Write inbound CRUD tests**

Create `internal/store/inbound_test.go`:

```go
package store

import (
	"testing"
)

func mustOpenTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) failed: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateAndGetInbound(t *testing.T) {
	s := mustOpenTestStore(t)

	ib := &Inbound{
		Tag:      "ss-12345",
		Protocol: "shadowsocks",
		Port:     12345,
		Settings: `{"method":"aes-256-gcm","password":"test123"}`,
	}
	if err := s.CreateInbound(ib); err != nil {
		t.Fatalf("CreateInbound error: %v", err)
	}
	if ib.ID == 0 {
		t.Fatal("expected ID to be set after create")
	}

	got, err := s.GetInbound(ib.ID)
	if err != nil {
		t.Fatalf("GetInbound error: %v", err)
	}
	if got.Tag != "ss-12345" {
		t.Errorf("tag = %q, want %q", got.Tag, "ss-12345")
	}
	if got.Protocol != "shadowsocks" {
		t.Errorf("protocol = %q, want %q", got.Protocol, "shadowsocks")
	}
	if got.Port != 12345 {
		t.Errorf("port = %d, want %d", got.Port, 12345)
	}
	if got.Settings != `{"method":"aes-256-gcm","password":"test123"}` {
		t.Errorf("settings = %q, want JSON", got.Settings)
	}
	if got.CreatedAt.IsZero() {
		t.Error("created_at should not be zero")
	}
}

func TestListInbounds(t *testing.T) {
	s := mustOpenTestStore(t)

	// Empty list
	list, err := s.ListInbounds()
	if err != nil {
		t.Fatalf("ListInbounds error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 inbounds, got %d", len(list))
	}

	// Add two
	s.CreateInbound(&Inbound{Tag: "ss-1001", Protocol: "shadowsocks", Port: 1001, Settings: "{}"})
	s.CreateInbound(&Inbound{Tag: "ss-1002", Protocol: "shadowsocks", Port: 1002, Settings: "{}"})

	list, err = s.ListInbounds()
	if err != nil {
		t.Fatalf("ListInbounds error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 inbounds, got %d", len(list))
	}
}

func TestUpdateInbound(t *testing.T) {
	s := mustOpenTestStore(t)

	ib := &Inbound{Tag: "ss-2000", Protocol: "shadowsocks", Port: 2000, Settings: `{"method":"aes-256-gcm","password":"old"}`}
	s.CreateInbound(ib)

	ib.Port = 2001
	ib.Settings = `{"method":"aes-256-gcm","password":"new"}`
	if err := s.UpdateInbound(ib); err != nil {
		t.Fatalf("UpdateInbound error: %v", err)
	}

	got, _ := s.GetInbound(ib.ID)
	if got.Port != 2001 {
		t.Errorf("port = %d, want 2001", got.Port)
	}
	if got.Settings != `{"method":"aes-256-gcm","password":"new"}` {
		t.Errorf("settings not updated")
	}
}

func TestDeleteInbound(t *testing.T) {
	s := mustOpenTestStore(t)

	ib := &Inbound{Tag: "ss-3000", Protocol: "shadowsocks", Port: 3000, Settings: "{}"}
	s.CreateInbound(ib)

	if err := s.DeleteInbound(ib.ID); err != nil {
		t.Fatalf("DeleteInbound error: %v", err)
	}

	_, err := s.GetInbound(ib.ID)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got: %v", err)
	}
}

func TestCreateDuplicateTag(t *testing.T) {
	s := mustOpenTestStore(t)

	ib1 := &Inbound{Tag: "ss-4000", Protocol: "shadowsocks", Port: 4000, Settings: "{}"}
	s.CreateInbound(ib1)

	ib2 := &Inbound{Tag: "ss-4000", Protocol: "shadowsocks", Port: 4001, Settings: "{}"}
	err := s.CreateInbound(ib2)
	if err == nil {
		t.Fatal("expected error for duplicate tag")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/store/ -v -count=1 -run TestCreate`

Expected: Compilation error — `Inbound` type and CRUD methods not defined.

- [ ] **Step 3: Implement inbound.go**

Create `internal/store/inbound.go`:

```go
package store

import (
	"errors"
	"fmt"
	"time"
)

// ErrNotFound is returned when a record does not exist.
var ErrNotFound = errors.New("not found")

// Inbound represents a proxy inbound configuration stored in the database.
type Inbound struct {
	ID        int64     `json:"id"`
	Tag       string    `json:"tag"`
	Protocol  string    `json:"protocol"`
	Port      uint16    `json:"port"`
	Settings  string    `json:"settings"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateInbound inserts a new inbound. Sets ib.ID on success.
func (s *Store) CreateInbound(ib *Inbound) error {
	result, err := s.db.Exec(
		`INSERT INTO inbounds (tag, protocol, port, settings) VALUES (?, ?, ?, ?)`,
		ib.Tag, ib.Protocol, ib.Port, ib.Settings,
	)
	if err != nil {
		return fmt.Errorf("failed to create inbound: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	ib.ID = id
	return nil
}

// GetInbound retrieves an inbound by ID. Returns ErrNotFound if not found.
func (s *Store) GetInbound(id int64) (*Inbound, error) {
	ib := &Inbound{}
	err := s.db.QueryRow(
		`SELECT id, tag, protocol, port, settings, created_at, updated_at FROM inbounds WHERE id = ?`,
		id,
	).Scan(&ib.ID, &ib.Tag, &ib.Protocol, &ib.Port, &ib.Settings, &ib.CreatedAt, &ib.UpdatedAt)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get inbound: %w", err)
	}
	return ib, nil
}

// ListInbounds returns all inbounds ordered by ID.
func (s *Store) ListInbounds() ([]*Inbound, error) {
	rows, err := s.db.Query(
		`SELECT id, tag, protocol, port, settings, created_at, updated_at FROM inbounds ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list inbounds: %w", err)
	}
	defer rows.Close()

	var inbounds []*Inbound
	for rows.Next() {
		ib := &Inbound{}
		if err := rows.Scan(&ib.ID, &ib.Tag, &ib.Protocol, &ib.Port, &ib.Settings, &ib.CreatedAt, &ib.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan inbound: %w", err)
		}
		inbounds = append(inbounds, ib)
	}
	return inbounds, rows.Err()
}

// UpdateInbound updates an existing inbound by ID.
func (s *Store) UpdateInbound(ib *Inbound) error {
	result, err := s.db.Exec(
		`UPDATE inbounds SET tag = ?, protocol = ?, port = ?, settings = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		ib.Tag, ib.Protocol, ib.Port, ib.Settings, ib.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update inbound: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteInbound removes an inbound by ID.
func (s *Store) DeleteInbound(id int64) error {
	result, err := s.db.Exec(`DELETE FROM inbounds WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete inbound: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/store/ -v -count=1 -race`

Expected: All tests PASS (store_test.go + inbound_test.go).

- [ ] **Step 5: Commit**

```bash
git add internal/store/inbound.go internal/store/inbound_test.go
git commit -m "feat(store): inbound model with CRUD operations"
```

---

### Task 4: Platform Detection

**Files:**
- Create: `internal/platform/platform.go`
- Create: `internal/platform/platform_test.go`

- [ ] **Step 1: Write platform tests**

Create `internal/platform/platform_test.go`:

```go
package platform

import (
	"runtime"
	"testing"
)

func TestDetectReturnsValidInfo(t *testing.T) {
	info := Detect()

	if info.OS == "" {
		t.Error("OS should not be empty")
	}
	if info.Arch == "" {
		t.Error("Arch should not be empty")
	}
	if info.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", info.OS, runtime.GOOS)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("Arch = %q, want %q", info.Arch, runtime.GOARCH)
	}
	if info.DataDir == "" {
		t.Error("DataDir should not be empty")
	}
	if info.BinPath == "" {
		t.Error("BinPath should not be empty")
	}
	if info.LogDir == "" {
		t.Error("LogDir should not be empty")
	}
}

func TestDetectInitSystemParsing(t *testing.T) {
	tests := []struct {
		name     string
		pid1     string
		expected string
	}{
		{"systemd", "systemd", "systemd"},
		{"openrc init", "init", "openrc"},
		{"openrc openrc-init", "openrc-init", "openrc"},
		{"unknown", "launchd", "unknown"},
		{"empty", "", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInitSystem(tt.pid1)
			if result != tt.expected {
				t.Errorf("parseInitSystem(%q) = %q, want %q", tt.pid1, result, tt.expected)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/platform/ -v -count=1`

Expected: Compilation error — package has no Go files.

- [ ] **Step 3: Implement platform.go**

Create `internal/platform/platform.go`:

```go
package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	InitSystemd = "systemd"
	InitOpenRC  = "openrc"
	InitUnknown = "unknown"
)

// Info holds detected platform information.
type Info struct {
	OS         string // runtime.GOOS
	Arch       string // runtime.GOARCH
	InitSystem string // "systemd", "openrc", or "unknown"
	DataDir    string // e.g. /usr/local/etc/sing-box
	BinPath    string // e.g. /usr/local/bin/sing-box
	LogDir     string // e.g. /var/log/sing-box
}

// Detect probes the current system and returns platform info.
func Detect() *Info {
	info := &Info{
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
		DataDir: "/usr/local/etc/sing-box",
		BinPath: "/usr/local/bin/sing-box",
		LogDir:  "/var/log/sing-box",
	}

	info.InitSystem = detectInitSystem()
	return info
}

func detectInitSystem() string {
	// Read PID 1 command name
	data, err := os.ReadFile("/proc/1/comm")
	if err != nil {
		return InitUnknown
	}
	return parseInitSystem(strings.TrimSpace(string(data)))
}

// parseInitSystem interprets the PID 1 process name.
// Exported for testing.
func parseInitSystem(pid1 string) string {
	switch {
	case pid1 == "systemd":
		return InitSystemd
	case pid1 == "init" || pid1 == "openrc-init":
		return InitOpenRC
	case pid1 == "":
		return InitUnknown
	default:
		return InitUnknown
	}
}

// DBPath returns the full path to the SQLite database file.
func (info *Info) DBPath() string {
	return filepath.Join(info.DataDir, "panel.db")
}

// ServiceName returns the systemd/openrc service name.
func (info *Info) ServiceName() string {
	return "sing-box"
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/platform/ -v -count=1`

Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/platform/platform.go internal/platform/platform_test.go
git commit -m "feat(platform): OS, arch, and init system detection"
```

---

### Task 5: Engine Lifecycle

**Files:**
- Create: `internal/engine/engine.go`
- Create: `internal/engine/build.go`
- Create: `internal/engine/build_test.go`

- [ ] **Step 1: Write build logic tests**

Create `internal/engine/build_test.go`:

```go
package engine

import (
	"testing"

	"github.com/cosaria/sing-box/internal/store"
)

func TestBuildShadowsocksInbound(t *testing.T) {
	ib := &store.Inbound{
		ID:       1,
		Tag:      "ss-12345",
		Protocol: "shadowsocks",
		Port:     12345,
		Settings: `{"method":"aes-256-gcm","password":"testpass"}`,
	}

	opt, err := buildInbound(ib)
	if err != nil {
		t.Fatalf("buildInbound error: %v", err)
	}
	if opt.Type != "shadowsocks" {
		t.Errorf("type = %q, want %q", opt.Type, "shadowsocks")
	}
	if opt.Tag != "ss-12345" {
		t.Errorf("tag = %q, want %q", opt.Tag, "ss-12345")
	}
}

func TestBuildShadowsocksInvalidJSON(t *testing.T) {
	ib := &store.Inbound{
		Protocol: "shadowsocks",
		Settings: `{invalid}`,
	}

	_, err := buildInbound(ib)
	if err == nil {
		t.Fatal("expected error for invalid JSON settings")
	}
}

func TestBuildUnsupportedProtocol(t *testing.T) {
	ib := &store.Inbound{
		Protocol: "unknown-proto",
		Settings: "{}",
	}

	_, err := buildInbound(ib)
	if err == nil {
		t.Fatal("expected error for unsupported protocol")
	}
}

func TestBuildOptionsEmpty(t *testing.T) {
	opts, err := buildOptions(nil)
	if err != nil {
		t.Fatalf("buildOptions(nil) error: %v", err)
	}
	if len(opts.Inbounds) != 0 {
		t.Errorf("expected 0 inbounds, got %d", len(opts.Inbounds))
	}
	if len(opts.Outbounds) != 1 {
		t.Fatalf("expected 1 outbound (direct), got %d", len(opts.Outbounds))
	}
	if opts.Outbounds[0].Type != "direct" {
		t.Errorf("outbound type = %q, want 'direct'", opts.Outbounds[0].Type)
	}
}

func TestBuildOptionsWithInbounds(t *testing.T) {
	inbounds := []*store.Inbound{
		{Tag: "ss-1000", Protocol: "shadowsocks", Port: 1000, Settings: `{"method":"aes-256-gcm","password":"p1"}`},
		{Tag: "ss-2000", Protocol: "shadowsocks", Port: 2000, Settings: `{"method":"chacha20-ietf-poly1305","password":"p2"}`},
	}

	opts, err := buildOptions(inbounds)
	if err != nil {
		t.Fatalf("buildOptions error: %v", err)
	}
	if len(opts.Inbounds) != 2 {
		t.Errorf("expected 2 inbounds, got %d", len(opts.Inbounds))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/engine/ -v -count=1`

Expected: Compilation error — no Go files in engine package.

- [ ] **Step 3: Implement build.go**

Create `internal/engine/build.go`:

```go
package engine

import (
	"encoding/json"
	"fmt"

	"github.com/cosaria/sing-box/internal/store"
	"github.com/sagernet/sing-box/option"
)

// shadowsocksSettings maps the JSON settings for a Shadowsocks inbound.
type shadowsocksSettings struct {
	Method   string `json:"method"`
	Password string `json:"password"`
}

// buildOptions creates sing-box option.Options from a list of store inbounds.
func buildOptions(inbounds []*store.Inbound) (option.Options, error) {
	opts := option.Options{
		Log: &option.LogOptions{
			Level: "info",
		},
		Outbounds: []option.Outbound{
			{Type: "direct", Tag: "direct"},
		},
	}

	for _, ib := range inbounds {
		singIb, err := buildInbound(ib)
		if err != nil {
			return opts, fmt.Errorf("failed to build inbound %q: %w", ib.Tag, err)
		}
		opts.Inbounds = append(opts.Inbounds, singIb)
	}
	return opts, nil
}

// buildInbound converts a store.Inbound to a sing-box option.Inbound.
func buildInbound(ib *store.Inbound) (option.Inbound, error) {
	switch ib.Protocol {
	case "shadowsocks":
		return buildShadowsocks(ib)
	default:
		return option.Inbound{}, fmt.Errorf("unsupported protocol: %s", ib.Protocol)
	}
}

func buildShadowsocks(ib *store.Inbound) (option.Inbound, error) {
	var ss shadowsocksSettings
	if err := json.Unmarshal([]byte(ib.Settings), &ss); err != nil {
		return option.Inbound{}, fmt.Errorf("invalid shadowsocks settings: %w", err)
	}

	return option.Inbound{
		Type: "shadowsocks",
		Tag:  ib.Tag,
		Options: &option.ShadowsocksInboundOptions{
			ListenOptions: option.ListenOptions{
				ListenPort: ib.Port,
			},
			Method:   ss.Method,
			Password: ss.Password,
		},
	}, nil
}
```

- [ ] **Step 4: Run build tests to verify they pass**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/engine/ -v -count=1 -race`

Expected: All 5 tests PASS.

- [ ] **Step 5: Implement engine.go**

Create `internal/engine/engine.go`:

```go
package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/cosaria/sing-box/internal/store"
	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
)

// Engine wraps the embedded sing-box instance with lifecycle management.
type Engine struct {
	store    *store.Store
	mu       sync.Mutex
	instance *box.Box
	running  bool
	startedAt time.Time
}

// New creates an Engine backed by the given store.
func New(s *store.Store) *Engine {
	return &Engine{store: s}
}

// Start loads inbounds from the store and starts the sing-box instance.
func (e *Engine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("engine already running")
	}

	inbounds, err := e.store.ListInbounds()
	if err != nil {
		return fmt.Errorf("failed to list inbounds: %w", err)
	}

	opts, err := buildOptions(inbounds)
	if err != nil {
		return fmt.Errorf("failed to build options: %w", err)
	}

	ctx := include.Context(context.Background())
	instance, err := box.New(box.Options{
		Context: ctx,
		Options: opts,
	})
	if err != nil {
		return fmt.Errorf("failed to create sing-box instance: %w", err)
	}

	if err := instance.Start(); err != nil {
		instance.Close()
		return fmt.Errorf("failed to start sing-box: %w", err)
	}

	e.instance = instance
	e.running = true
	e.startedAt = time.Now()
	slog.Info("engine started", "inbounds", len(inbounds))
	return nil
}

// Stop gracefully shuts down the sing-box instance.
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}

	if err := e.instance.Close(); err != nil {
		return fmt.Errorf("failed to close sing-box: %w", err)
	}

	e.instance = nil
	e.running = false
	slog.Info("engine stopped")
	return nil
}

// Reload stops the current instance and starts a new one with fresh config from store.
func (e *Engine) Reload() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Close existing instance if running
	if e.running && e.instance != nil {
		if err := e.instance.Close(); err != nil {
			slog.Warn("failed to close old instance during reload", "error", err)
		}
		e.instance = nil
		e.running = false
	}

	// Build new config
	inbounds, err := e.store.ListInbounds()
	if err != nil {
		return fmt.Errorf("failed to list inbounds: %w", err)
	}

	opts, err := buildOptions(inbounds)
	if err != nil {
		return fmt.Errorf("failed to build options: %w", err)
	}

	ctx := include.Context(context.Background())
	instance, err := box.New(box.Options{
		Context: ctx,
		Options: opts,
	})
	if err != nil {
		return fmt.Errorf("failed to create new sing-box instance: %w", err)
	}

	if err := instance.Start(); err != nil {
		instance.Close()
		return fmt.Errorf("failed to start new sing-box instance: %w", err)
	}

	e.instance = instance
	e.running = true
	e.startedAt = time.Now()
	slog.Info("engine reloaded", "inbounds", len(inbounds))
	return nil
}

// Running returns whether the engine is currently active.
func (e *Engine) Running() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

// StartedAt returns when the engine last started.
func (e *Engine) StartedAt() time.Time {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.startedAt
}
```

- [ ] **Step 6: Verify engine compiles**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go build ./internal/engine/`

Expected: Compiles without errors.

- [ ] **Step 7: Commit**

```bash
git add internal/engine/engine.go internal/engine/build.go internal/engine/build_test.go
git commit -m "feat(engine): sing-box lifecycle with Start/Stop/Reload and config builder"
```

---

### Task 6: Service Management

**Files:**
- Create: `internal/service/service.go`
- Create: `internal/service/systemd.go`
- Create: `internal/service/openrc.go`
- Create: `internal/service/service_test.go`

- [ ] **Step 1: Write service template tests**

Create `internal/service/service_test.go`:

```go
package service

import (
	"strings"
	"testing"
)

func TestSystemdUnit(t *testing.T) {
	content := systemdUnitContent("/usr/local/bin/sing-box", "/usr/local/etc/sing-box")
	if !strings.Contains(content, "ExecStart=/usr/local/bin/sing-box serve") {
		t.Error("unit should contain ExecStart with serve command")
	}
	if !strings.Contains(content, "--data-dir /usr/local/etc/sing-box") {
		t.Error("unit should contain --data-dir flag")
	}
	if !strings.Contains(content, "[Unit]") {
		t.Error("unit should have [Unit] section")
	}
}

func TestOpenRCInit(t *testing.T) {
	content := openrcInitContent("/usr/local/bin/sing-box", "/usr/local/etc/sing-box")
	if !strings.Contains(content, "command=\"/usr/local/bin/sing-box\"") {
		t.Error("init script should contain command path")
	}
	if !strings.Contains(content, "command_args=\"serve") {
		t.Error("init script should contain serve command")
	}
}

func TestNewManagerSystemd(t *testing.T) {
	m := NewManager("systemd")
	if m == nil {
		t.Fatal("NewManager(systemd) returned nil")
	}
}

func TestNewManagerOpenRC(t *testing.T) {
	m := NewManager("openrc")
	if m == nil {
		t.Fatal("NewManager(openrc) returned nil")
	}
}

func TestNewManagerUnknown(t *testing.T) {
	m := NewManager("unknown")
	if m != nil {
		t.Fatal("NewManager(unknown) should return nil")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/service/ -v -count=1`

Expected: Compilation error.

- [ ] **Step 3: Implement service.go**

Create `internal/service/service.go`:

```go
package service

// Manager abstracts system service operations (systemd/OpenRC).
type Manager interface {
	// Install writes the service unit/init script and enables it.
	Install(binPath, dataDir string) error
	// Uninstall stops and removes the service.
	Uninstall() error
	// Start starts the service.
	Start() error
	// Stop stops the service.
	Stop() error
	// Restart restarts the service.
	Restart() error
	// Status returns the service state: "running", "stopped", or "not-installed".
	Status() (string, error)
}

// NewManager returns a Manager for the given init system, or nil if unsupported.
func NewManager(initSystem string) Manager {
	switch initSystem {
	case "systemd":
		return &systemdManager{}
	case "openrc":
		return &openrcManager{}
	default:
		return nil
	}
}
```

- [ ] **Step 4: Implement systemd.go**

Create `internal/service/systemd.go`:

```go
package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const systemdUnitPath = "/etc/systemd/system/sing-box.service"

type systemdManager struct{}

func systemdUnitContent(binPath, dataDir string) string {
	return fmt.Sprintf(`[Unit]
Description=sing-box Panel Service
After=network.target nss-lookup.target

[Service]
Type=simple
ExecStart=%s serve --data-dir %s
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
`, binPath, dataDir)
}

func (m *systemdManager) Install(binPath, dataDir string) error {
	content := systemdUnitContent(binPath, dataDir)
	if err := os.WriteFile(systemdUnitPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write unit file: %w", err)
	}
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}
	if err := exec.Command("systemctl", "enable", "sing-box").Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}
	return nil
}

func (m *systemdManager) Uninstall() error {
	exec.Command("systemctl", "stop", "sing-box").Run()
	exec.Command("systemctl", "disable", "sing-box").Run()
	os.Remove(systemdUnitPath)
	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}

func (m *systemdManager) Start() error {
	return exec.Command("systemctl", "start", "sing-box").Run()
}

func (m *systemdManager) Stop() error {
	return exec.Command("systemctl", "stop", "sing-box").Run()
}

func (m *systemdManager) Restart() error {
	return exec.Command("systemctl", "restart", "sing-box").Run()
}

func (m *systemdManager) Status() (string, error) {
	out, err := exec.Command("systemctl", "is-active", "sing-box").Output()
	status := strings.TrimSpace(string(out))
	if err != nil {
		if status == "inactive" {
			return "stopped", nil
		}
		return "not-installed", nil
	}
	if status == "active" {
		return "running", nil
	}
	return "stopped", nil
}
```

- [ ] **Step 5: Implement openrc.go**

Create `internal/service/openrc.go`:

```go
package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const openrcInitPath = "/etc/init.d/sing-box"

type openrcManager struct{}

func openrcInitContent(binPath, dataDir string) string {
	return fmt.Sprintf(`#!/sbin/openrc-run

description="sing-box Panel Service"

command="%s"
command_args="serve --data-dir %s"
command_background=true
pidfile="/run/sing-box.pid"

depend() {
	need net
	after firewall
}
`, binPath, dataDir)
}

func (m *openrcManager) Install(binPath, dataDir string) error {
	content := openrcInitContent(binPath, dataDir)
	if err := os.WriteFile(openrcInitPath, []byte(content), 0755); err != nil {
		return fmt.Errorf("failed to write init script: %w", err)
	}
	if err := exec.Command("rc-update", "add", "sing-box", "default").Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}
	return nil
}

func (m *openrcManager) Uninstall() error {
	exec.Command("rc-service", "sing-box", "stop").Run()
	exec.Command("rc-update", "del", "sing-box", "default").Run()
	os.Remove(openrcInitPath)
	return nil
}

func (m *openrcManager) Start() error {
	return exec.Command("rc-service", "sing-box", "start").Run()
}

func (m *openrcManager) Stop() error {
	return exec.Command("rc-service", "sing-box", "stop").Run()
}

func (m *openrcManager) Restart() error {
	return exec.Command("rc-service", "sing-box", "restart").Run()
}

func (m *openrcManager) Status() (string, error) {
	out, err := exec.Command("rc-service", "sing-box", "status").Output()
	status := strings.TrimSpace(string(out))
	if err != nil {
		return "not-installed", nil
	}
	if strings.Contains(status, "started") {
		return "running", nil
	}
	return "stopped", nil
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/service/ -v -count=1`

Expected: All 5 tests PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/service/
git commit -m "feat(service): systemd and OpenRC service management"
```

---

### Task 7: HTTP API Server + All Handlers

**Files:**
- Create: `internal/api/server.go`
- Create: `internal/api/middleware.go`
- Create: `internal/api/inbound.go`
- Create: `internal/api/engine.go`
- Create: `internal/api/api_test.go`

- [ ] **Step 1: Write API handler tests**

Create `internal/api/api_test.go`:

```go
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cosaria/sing-box/internal/store"
)

// mockEngine implements engine control for testing without sing-box dependency.
type mockEngine struct {
	running   bool
	reloaded  int
	startErr  error
	reloadErr error
}

func (m *mockEngine) Start() error   { m.running = true; return m.startErr }
func (m *mockEngine) Stop() error    { m.running = false; return nil }
func (m *mockEngine) Reload() error  { m.reloaded++; return m.reloadErr }
func (m *mockEngine) Running() bool  { return m.running }

func setupTestServer(t *testing.T) (*Server, *store.Store, *mockEngine) {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	eng := &mockEngine{}
	srv := NewServer(eng, st, nil, "127.0.0.1:0", "test-token")
	return srv, st, eng
}

func TestGetStatusUnauthorized(t *testing.T) {
	srv, _, _ := setupTestServer(t)
	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestGetStatusAuthorized(t *testing.T) {
	srv, _, eng := setupTestServer(t)
	eng.running = true
	req := httptest.NewRequest("GET", "/api/status", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["engine"] != "running" {
		t.Errorf("engine = %v, want 'running'", resp["engine"])
	}
}

func TestCreateInbound(t *testing.T) {
	srv, _, eng := setupTestServer(t)
	body := `{"protocol":"shadowsocks","port":12345,"settings":{"method":"aes-256-gcm","password":"test"}}`
	req := httptest.NewRequest("POST", "/api/inbounds", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["tag"] != "ss-12345" {
		t.Errorf("tag = %v, want 'ss-12345'", resp["tag"])
	}
	if eng.reloaded != 1 {
		t.Errorf("engine.Reload() called %d times, want 1", eng.reloaded)
	}
}

func TestListInbounds(t *testing.T) {
	srv, st, _ := setupTestServer(t)
	st.CreateInbound(&store.Inbound{Tag: "ss-1000", Protocol: "shadowsocks", Port: 1000, Settings: "{}"})
	st.CreateInbound(&store.Inbound{Tag: "ss-2000", Protocol: "shadowsocks", Port: 2000, Settings: "{}"})

	req := httptest.NewRequest("GET", "/api/inbounds", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp []map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 inbounds, got %d", len(resp))
	}
}

func TestGetInbound(t *testing.T) {
	srv, st, _ := setupTestServer(t)
	ib := &store.Inbound{Tag: "ss-5000", Protocol: "shadowsocks", Port: 5000, Settings: "{}"}
	st.CreateInbound(ib)

	req := httptest.NewRequest("GET", "/api/inbounds/1", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestGetInboundNotFound(t *testing.T) {
	srv, _, _ := setupTestServer(t)
	req := httptest.NewRequest("GET", "/api/inbounds/999", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteInbound(t *testing.T) {
	srv, st, eng := setupTestServer(t)
	ib := &store.Inbound{Tag: "ss-6000", Protocol: "shadowsocks", Port: 6000, Settings: "{}"}
	st.CreateInbound(ib)

	req := httptest.NewRequest("DELETE", "/api/inbounds/1", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if eng.reloaded != 1 {
		t.Errorf("engine.Reload() called %d times, want 1", eng.reloaded)
	}
}

func TestReloadEndpoint(t *testing.T) {
	srv, _, eng := setupTestServer(t)
	req := httptest.NewRequest("POST", "/api/reload", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if eng.reloaded != 1 {
		t.Errorf("engine.Reload() called %d times, want 1", eng.reloaded)
	}
}
```

- [ ] **Step 2: Implement middleware.go**

Create `internal/api/middleware.go`:

```go
package api

import (
	"net/http"
	"strings"
)

// tokenAuth returns middleware that validates Bearer token authentication.
func tokenAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			if strings.TrimPrefix(auth, "Bearer ") != token {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 3: Implement server.go**

Create `internal/api/server.go`:

```go
package api

import (
	"context"
	"net"
	"net/http"

	"github.com/cosaria/sing-box/internal/store"
	"github.com/go-chi/chi/v5"
)

// EngineController defines the engine operations the API needs.
type EngineController interface {
	Start() error
	Stop() error
	Reload() error
	Running() bool
}

// Server is the HTTP API server.
type Server struct {
	engine  EngineController
	store   *store.Store
	router  chi.Router
	httpSrv *http.Server
	token   string
}

// NewServer creates an API server wired to the given engine and store.
// svc may be nil if service management is unavailable on this platform.
func NewServer(engine EngineController, st *store.Store, svc any, listenAddr, token string) *Server {
	s := &Server{
		engine: engine,
		store:  st,
		token:  token,
	}

	r := chi.NewRouter()
	r.Use(tokenAuth(token))

	// Engine endpoints
	r.Get("/api/status", s.handleStatus)
	r.Post("/api/reload", s.handleReload)

	// Inbound CRUD
	r.Get("/api/inbounds", s.handleListInbounds)
	r.Post("/api/inbounds", s.handleCreateInbound)
	r.Get("/api/inbounds/{id}", s.handleGetInbound)
	r.Put("/api/inbounds/{id}", s.handleUpdateInbound)
	r.Delete("/api/inbounds/{id}", s.handleDeleteInbound)

	s.router = r
	s.httpSrv = &http.Server{
		Addr:    listenAddr,
		Handler: r,
	}
	return s
}

// Start begins listening for HTTP requests.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.httpSrv.Addr)
	if err != nil {
		return err
	}
	go s.httpSrv.Serve(ln)
	return nil
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}
```

- [ ] **Step 4: Implement engine.go (engine/status handlers)**

Create `internal/api/engine.go`:

```go
package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := "stopped"
	if s.engine.Running() {
		status = "running"
	}

	inbounds, _ := s.store.ListInbounds()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"engine":   status,
		"inbounds": len(inbounds),
	})
}

func (s *Server) handleReload(w http.ResponseWriter, r *http.Request) {
	if err := s.engine.Reload(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "reloaded"})
}
```

- [ ] **Step 5: Implement inbound.go (CRUD handlers)**

Create `internal/api/inbound.go`:

```go
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cosaria/sing-box/internal/store"
	"github.com/go-chi/chi/v5"
)

type createInboundRequest struct {
	Protocol string          `json:"protocol"`
	Port     uint16          `json:"port"`
	Settings json.RawMessage `json:"settings"`
}

type updateInboundRequest struct {
	Port     *uint16          `json:"port,omitempty"`
	Settings *json.RawMessage `json:"settings,omitempty"`
}

func (s *Server) handleListInbounds(w http.ResponseWriter, r *http.Request) {
	inbounds, err := s.store.ListInbounds()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Return empty array instead of null
	if inbounds == nil {
		inbounds = []*store.Inbound{}
	}
	writeJSON(w, http.StatusOK, inbounds)
}

func (s *Server) handleCreateInbound(w http.ResponseWriter, r *http.Request) {
	var req createInboundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Protocol == "" {
		writeError(w, http.StatusBadRequest, "protocol is required")
		return
	}
	if req.Port == 0 {
		writeError(w, http.StatusBadRequest, "port is required")
		return
	}

	settingsStr := "{}"
	if req.Settings != nil {
		settingsStr = string(req.Settings)
	}

	tag := fmt.Sprintf("ss-%d", req.Port)
	if req.Protocol != "shadowsocks" {
		tag = fmt.Sprintf("%s-%d", req.Protocol, req.Port)
	}

	ib := &store.Inbound{
		Tag:      tag,
		Protocol: req.Protocol,
		Port:     req.Port,
		Settings: settingsStr,
	}

	if err := s.store.CreateInbound(ib); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	// Reload engine to pick up new inbound
	s.engine.Reload()

	writeJSON(w, http.StatusCreated, ib)
}

func (s *Server) handleGetInbound(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	ib, err := s.store.GetInbound(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "inbound not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ib)
}

func (s *Server) handleUpdateInbound(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	ib, err := s.store.GetInbound(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "inbound not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var req updateInboundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Port != nil {
		ib.Port = *req.Port
	}
	if req.Settings != nil {
		ib.Settings = string(*req.Settings)
	}

	if err := s.store.UpdateInbound(ib); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.engine.Reload()
	writeJSON(w, http.StatusOK, ib)
}

func (s *Server) handleDeleteInbound(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := s.store.DeleteInbound(id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "inbound not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.engine.Reload()
	w.WriteHeader(http.StatusNoContent)
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
```

- [ ] **Step 6: Run API tests to verify they pass**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/api/ -v -count=1 -race`

Expected: All 8 tests PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/api/
git commit -m "feat(api): HTTP API server with inbound CRUD, auth, and engine control"
```

---

### Task 8: Main Entry Point + Integration

**Files:**
- Modify: `cmd/sing-box/main.go` (complete rewrite from M0 prototype)

- [ ] **Step 1: Implement cobra CLI with serve command**

Rewrite `cmd/sing-box/main.go`:

```go
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cosaria/sing-box/internal/api"
	"github.com/cosaria/sing-box/internal/engine"
	"github.com/cosaria/sing-box/internal/platform"
	"github.com/cosaria/sing-box/internal/service"
	"github.com/cosaria/sing-box/internal/store"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "sing-box",
		Short: "sing-box 管理面板",
	}

	rootCmd.AddCommand(serveCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func serveCmd() *cobra.Command {
	var (
		listenAddr string
		dataDir    string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "启动守护进程（HTTP API + sing-box 引擎）",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(listenAddr, dataDir)
		},
	}

	plat := platform.Detect()
	cmd.Flags().StringVar(&listenAddr, "listen", "127.0.0.1:9090", "API 监听地址")
	cmd.Flags().StringVar(&dataDir, "data-dir", plat.DataDir, "数据目录")

	return cmd
}

func runServe(listenAddr, dataDir string) error {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("无法创建数据目录: %w", err)
	}

	// Open store
	dbPath := dataDir + "/panel.db"
	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("无法打开数据库: %w", err)
	}
	defer st.Close()

	// Ensure API token exists
	token, err := st.GetSetting("api_token")
	if err != nil {
		return fmt.Errorf("无法读取 API token: %w", err)
	}
	if token == "" {
		token = generateToken()
		if err := st.SetSetting("api_token", token); err != nil {
			return fmt.Errorf("无法保存 API token: %w", err)
		}
	}

	// Detect platform
	plat := platform.Detect()
	var svcMgr service.Manager
	if mgr := service.NewManager(plat.InitSystem); mgr != nil {
		svcMgr = mgr
	}

	// Create and start engine
	eng := engine.New(st)
	if err := eng.Start(); err != nil {
		slog.Warn("引擎启动失败（可能没有配置）", "error", err)
	}

	// Create and start API server
	srv := api.NewServer(eng, st, svcMgr, listenAddr, token)
	if err := srv.Start(); err != nil {
		return fmt.Errorf("无法启动 API 服务: %w", err)
	}

	slog.Info("sing-box 面板已启动",
		"listen", listenAddr,
		"data_dir", dataDir,
		"init_system", plat.InitSystem,
	)
	fmt.Printf("\nAPI Token: %s\n", token)
	fmt.Printf("API 地址: http://%s\n\n", listenAddr)

	// Wait for signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {
		sig := <-sigCh
		switch sig {
		case syscall.SIGHUP:
			slog.Info("收到 SIGHUP，重载引擎...")
			if err := eng.Reload(); err != nil {
				slog.Error("重载失败", "error", err)
			}
		case syscall.SIGINT, syscall.SIGTERM:
			slog.Info("正在关闭...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			srv.Shutdown(shutdownCtx)
			eng.Stop()
			slog.Info("已关闭")
			return nil
		}
	}
}

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go build -o sing-box-m1 ./cmd/sing-box/`

Expected: Binary `sing-box-m1` built successfully.

- [ ] **Step 3: Verify help output**

Run: `./sing-box-m1 --help`

Expected output should show `serve` subcommand.

Run: `./sing-box-m1 serve --help`

Expected output should show `--listen` and `--data-dir` flags.

- [ ] **Step 4: Smoke test with temporary data dir**

```bash
mkdir -p /tmp/sing-box-test
./sing-box-m1 serve --data-dir /tmp/sing-box-test --listen 127.0.0.1:19090 &
SERVE_PID=$!
sleep 2

# Get token from DB
TOKEN=$(sqlite3 /tmp/sing-box-test/panel.db "SELECT value FROM settings WHERE key='api_token'")

# Test status
curl -s -H "Authorization: Bearer $TOKEN" http://127.0.0.1:19090/api/status | python3 -m json.tool

# Test create inbound
curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"protocol":"shadowsocks","port":18388,"settings":{"method":"aes-256-gcm","password":"test123"}}' \
  http://127.0.0.1:19090/api/inbounds | python3 -m json.tool

# Test list inbounds
curl -s -H "Authorization: Bearer $TOKEN" http://127.0.0.1:19090/api/inbounds | python3 -m json.tool

# Test unauthorized
curl -s http://127.0.0.1:19090/api/status

# Cleanup
kill $SERVE_PID
rm -rf /tmp/sing-box-test
```

Expected:
- `/api/status` returns `{"engine":"running","inbounds":0}` then `{"engine":"running","inbounds":1}` after create
- Create returns 201 with `"tag":"ss-18388"`
- List returns array with 1 inbound
- Unauthorized returns 401

- [ ] **Step 5: Update .gitignore and commit**

Add `sing-box-m1` to `.gitignore`, then:

```bash
git add cmd/sing-box/main.go .gitignore
git commit -m "feat: M1 — core backend with HTTP API, engine lifecycle, and service management"
```

---

### Task 9: Run All Tests + Final Verification

- [ ] **Step 1: Run full test suite**

```bash
cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/... -v -count=1 -race
```

Expected: All tests across store, platform, engine, service, and api packages PASS.

- [ ] **Step 2: Run go vet and staticcheck**

```bash
go vet ./...
```

Expected: No issues.

- [ ] **Step 3: Verify cross-compilation**

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /dev/null ./cmd/sing-box/
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /dev/null ./cmd/sing-box/
```

Expected: Both compile without errors.

- [ ] **Step 4: Final commit if any fixes needed**

If any fixes were made during verification, commit them:

```bash
git add -A
git commit -m "fix: M1 test and vet fixes"
```

---

## Summary

| Package | Responsibility | Key API |
|---------|---------------|---------|
| `store` | SQLite + migrations + inbound CRUD | `Open()`, `CreateInbound()`, `ListInbounds()`, etc. |
| `platform` | OS/arch/init-system detection | `Detect() *Info` |
| `engine` | sing-box lifecycle | `New()`, `Start()`, `Stop()`, `Reload()` |
| `service` | systemd/OpenRC management | `Manager` interface, `NewManager()` |
| `api` | HTTP API (chi/v5) | `NewServer()`, `Start()`, `Shutdown()` |

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/status` | Engine status + inbound count |
| POST | `/api/reload` | Hot-reload engine config |
| GET | `/api/inbounds` | List all inbounds |
| POST | `/api/inbounds` | Create inbound (triggers reload) |
| GET | `/api/inbounds/{id}` | Get inbound by ID |
| PUT | `/api/inbounds/{id}` | Update inbound (triggers reload) |
| DELETE | `/api/inbounds/{id}` | Delete inbound (triggers reload) |

All endpoints require `Authorization: Bearer {token}` header.
