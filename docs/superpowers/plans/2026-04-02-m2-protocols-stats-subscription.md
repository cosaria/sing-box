# M2: Protocol Framework + Traffic Stats + Subscription

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add multi-protocol support (Shadowsocks SS2022, VLESS-REALITY, Trojan), per-inbound traffic stats, and a subscription endpoint to the sing-box panel.

**Architecture:** Protocol interface + registry pattern for extensibility. Custom ConnectionTracker registered on sing-box Router for zero-overhead traffic counting. Subscription endpoint generates Base64-encoded share URLs from all inbounds.

**Tech Stack:** Go 1.26, sing-box v1.13.5 (embedded), chi/v5, modernc.org/sqlite, crypto/ecdh (X25519)

**Module path:** `github.com/233boy/sing-box`

---

## File Structure

```
internal/
├── protocol/
│   ├── protocol.go          # Protocol interface + registry + helpers (generateUUID, etc.)
│   ├── protocol_test.go     # Registry tests
│   ├── shadowsocks.go       # Shadowsocks SS2022 implementation
│   ├── shadowsocks_test.go  # Shadowsocks tests
│   ├── vless.go             # VLESS-REALITY implementation
│   ├── vless_test.go        # VLESS tests
│   ├── trojan.go            # Trojan implementation
│   ��── trojan_test.go       # Trojan tests
├── stats/
│   ├── tracker.go           # ConnectionTracker + per-inbound counters
│   ├── tracker_test.go      # Tracker unit tests
│   ├── collector.go         # Periodic DB writer
│   └── collector_test.go    # Collector tests
├── store/
│   ├── store.go             # (modify) Add migration 2: traffic_logs
│   └── traffic.go           # Traffic log queries
├── engine/
│   ├── build.go             # (modify) Use protocol registry
│   └── engine.go            # (modify) Register tracker, expose TrafficSnapshot
├── api/
│   ├── server.go            # (modify) Route groups, stats + subscription routes
│   ├── inbound.go           # (modify) DefaultSettings, merge, tag generation
│   ├── stats.go             # Stats API handlers
│   └── subscription.go      # Subscription endpoint handler
└── cmd/sing-box/main.go     # (modify) Wire collector, sub_token
```

## Dependency Graph

```
Group A (no deps):          protocol/protocol.go
Group B (needs A, parallel): shadowsocks.go ║ vless.go ║ trojan.go
Group C (needs B):          engine/build.go refactor + api/inbound.go refactor
Group D (no deps):          store migration + traffic.go
Group E (needs D):          stats/tracker.go + stats/collector.go
Group F (needs C+E):        engine.go tracker integration
Group G (needs F):          api/stats.go + api/subscription.go + server.go
Group H (needs G):          main.go wiring
Group I (final):            full verification
```

---

### Task 1: Protocol Interface + Registry

**Files:**
- Create: `internal/protocol/protocol.go`
- Create: `internal/protocol/protocol_test.go`

- [ ] **Step 1: Write registry tests**

Create `internal/protocol/protocol_test.go`:

```go
package protocol

import (
	"testing"

	"github.com/233boy/sing-box/internal/store"
	"github.com/sagernet/sing-box/option"
)

// mockProtocol is a minimal Protocol implementation for testing the registry.
type mockProtocol struct{}

func (m *mockProtocol) Name() string        { return "mock" }
func (m *mockProtocol) DisplayName() string  { return "Mock" }
func (m *mockProtocol) DefaultSettings(port uint16) (string, error) {
	return `{"key":"value"}`, nil
}
func (m *mockProtocol) BuildInbound(ib *store.Inbound) (option.Inbound, error) {
	return option.Inbound{Type: "mock", Tag: ib.Tag}, nil
}
func (m *mockProtocol) GenerateURL(ib *store.Inbound, host string) string {
	return "mock://" + host
}

func TestRegisterAndGet(t *testing.T) {
	// Reset registry for test isolation
	registry = map[string]Protocol{}

	Register(&mockProtocol{})

	p := Get("mock")
	if p == nil {
		t.Fatal("Get(mock) returned nil")
	}
	if p.Name() != "mock" {
		t.Errorf("Name() = %q, want %q", p.Name(), "mock")
	}
}

func TestGetUnknown(t *testing.T) {
	registry = map[string]Protocol{}

	p := Get("nonexistent")
	if p != nil {
		t.Fatal("Get(nonexistent) should return nil")
	}
}

func TestAll(t *testing.T) {
	registry = map[string]Protocol{}

	Register(&mockProtocol{})
	all := All()
	if len(all) != 1 {
		t.Fatalf("All() returned %d protocols, want 1", len(all))
	}
}

func TestGenerateUUID(t *testing.T) {
	id := GenerateUUID()
	if len(id) != 36 {
		t.Errorf("UUID length = %d, want 36", len(id))
	}
	// Check format: 8-4-4-4-12
	if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		t.Errorf("invalid UUID format: %s", id)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/protocol/ -v -count=1`

Expected: Compilation error — package has no Go files.

- [ ] **Step 3: Implement protocol.go**

Create `internal/protocol/protocol.go`:

```go
package protocol

import (
	"crypto/rand"
	"fmt"

	"github.com/233boy/sing-box/internal/store"
	"github.com/sagernet/sing-box/option"
)

// Protocol defines the contract every proxy protocol must implement.
type Protocol interface {
	Name() string
	DisplayName() string
	DefaultSettings(port uint16) (string, error)
	BuildInbound(ib *store.Inbound) (option.Inbound, error)
	GenerateURL(ib *store.Inbound, host string) string
}

var registry = map[string]Protocol{}

// Register adds a protocol to the global registry.
func Register(p Protocol) {
	registry[p.Name()] = p
}

// Get returns the protocol for the given name, or nil if not found.
func Get(name string) Protocol {
	return registry[name]
}

// All returns all registered protocols.
func All() []Protocol {
	out := make([]Protocol, 0, len(registry))
	for _, p := range registry {
		out = append(out, p)
	}
	return out
}

// GenerateUUID creates a random UUID v4 string.
func GenerateUUID() string {
	var uuid [16]byte
	rand.Read(uuid[:])
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant 1
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		uuid[0], uuid[1], uuid[2], uuid[3],
		uuid[4], uuid[5],
		uuid[6], uuid[7],
		uuid[8], uuid[9],
		uuid[10], uuid[11], uuid[12], uuid[13], uuid[14], uuid[15])
}

// GenerateShortID creates an 8-character hex string for Reality short_id.
func GenerateShortID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%02x%02x%02x%02x", b[0], b[1], b[2], b[3])
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/protocol/ -v -count=1`

Expected: All 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/protocol/protocol.go internal/protocol/protocol_test.go
git commit -m "feat(protocol): Protocol interface, registry, and UUID helpers"
```

---

### Task 2: Shadowsocks Protocol Implementation

**Files:**
- Create: `internal/protocol/shadowsocks.go`
- Create: `internal/protocol/shadowsocks_test.go`

- [ ] **Step 1: Write Shadowsocks tests**

Create `internal/protocol/shadowsocks_test.go`:

```go
package protocol

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/233boy/sing-box/internal/store"
)

func TestShadowsocksDefaultSettings(t *testing.T) {
	ss := &Shadowsocks{}
	settingsJSON, err := ss.DefaultSettings(8388)
	if err != nil {
		t.Fatalf("DefaultSettings error: %v", err)
	}

	var s shadowsocksSettings
	if err := json.Unmarshal([]byte(settingsJSON), &s); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if s.Method != "2022-blake3-aes-128-gcm" {
		t.Errorf("method = %q, want 2022-blake3-aes-128-gcm", s.Method)
	}
	if len(s.Password) != 36 {
		t.Errorf("password length = %d, want 36 (UUID)", len(s.Password))
	}
}

func TestShadowsocksBuildInbound(t *testing.T) {
	ss := &Shadowsocks{}
	ib := &store.Inbound{
		Tag:      "ss-8388",
		Protocol: "shadowsocks",
		Port:     8388,
		Settings: `{"method":"2022-blake3-aes-128-gcm","password":"550e8400-e29b-41d4-a716-446655440000"}`,
	}

	opt, err := ss.BuildInbound(ib)
	if err != nil {
		t.Fatalf("BuildInbound error: %v", err)
	}
	if opt.Type != "shadowsocks" {
		t.Errorf("type = %q, want shadowsocks", opt.Type)
	}
	if opt.Tag != "ss-8388" {
		t.Errorf("tag = %q, want ss-8388", opt.Tag)
	}
}

func TestShadowsocksBuildInboundInvalidJSON(t *testing.T) {
	ss := &Shadowsocks{}
	ib := &store.Inbound{Settings: `{invalid}`}

	_, err := ss.BuildInbound(ib)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestShadowsocksGenerateURL(t *testing.T) {
	ss := &Shadowsocks{}
	ib := &store.Inbound{
		Tag:      "ss-8388",
		Port:     8388,
		Settings: `{"method":"2022-blake3-aes-128-gcm","password":"550e8400-e29b-41d4-a716-446655440000"}`,
	}

	url := ss.GenerateURL(ib, "1.2.3.4")
	if !strings.HasPrefix(url, "ss://") {
		t.Errorf("URL should start with ss://, got %q", url)
	}
	if !strings.Contains(url, "@1.2.3.4:8388") {
		t.Errorf("URL should contain @1.2.3.4:8388, got %q", url)
	}
	if !strings.HasSuffix(url, "#ss-8388") {
		t.Errorf("URL should end with #ss-8388, got %q", url)
	}

	// Verify base64 part is decodable
	parts := strings.SplitN(strings.TrimPrefix(url, "ss://"), "@", 2)
	decoded, err := base64.URLEncoding.DecodeString(parts[0])
	if err != nil {
		// Try standard encoding
		decoded, err = base64.StdEncoding.DecodeString(parts[0])
		if err != nil {
			t.Fatalf("cannot decode base64 part: %v", err)
		}
	}
	if !strings.Contains(string(decoded), "2022-blake3-aes-128-gcm") {
		t.Errorf("decoded part should contain method, got %q", string(decoded))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/protocol/ -v -count=1 -run TestShadowsocks`

Expected: Compilation error — `Shadowsocks` type not defined.

- [ ] **Step 3: Implement shadowsocks.go**

Create `internal/protocol/shadowsocks.go`:

```go
package protocol

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/233boy/sing-box/internal/store"
	"github.com/sagernet/sing-box/option"
)

func init() {
	Register(&Shadowsocks{})
}

type shadowsocksSettings struct {
	Method   string `json:"method"`
	Password string `json:"password"`
}

// Shadowsocks implements Protocol for Shadowsocks (SS2022).
type Shadowsocks struct{}

func (s *Shadowsocks) Name() string        { return "shadowsocks" }
func (s *Shadowsocks) DisplayName() string  { return "Shadowsocks" }

func (s *Shadowsocks) DefaultSettings(port uint16) (string, error) {
	settings := shadowsocksSettings{
		Method:   "2022-blake3-aes-128-gcm",
		Password: GenerateUUID(),
	}
	b, err := json.Marshal(settings)
	return string(b), err
}

func (s *Shadowsocks) BuildInbound(ib *store.Inbound) (option.Inbound, error) {
	var ss shadowsocksSettings
	if err := json.Unmarshal([]byte(ib.Settings), &ss); err != nil {
		return option.Inbound{}, fmt.Errorf("invalid shadowsocks settings: %w", err)
	}

	// SS2022 requires base64-encoded key; UUID provides 16 raw bytes for 128-bit key
	password := ss.Password
	if isSSAEAD2022(ss.Method) {
		password = uuidToBase64Key(ss.Password)
	}

	return option.Inbound{
		Type: "shadowsocks",
		Tag:  ib.Tag,
		Options: &option.ShadowsocksInboundOptions{
			ListenOptions: option.ListenOptions{
				ListenPort: ib.Port,
			},
			Method:   ss.Method,
			Password: password,
		},
	}, nil
}

func (s *Shadowsocks) GenerateURL(ib *store.Inbound, host string) string {
	var ss shadowsocksSettings
	if err := json.Unmarshal([]byte(ib.Settings), &ss); err != nil {
		return ""
	}

	password := ss.Password
	if isSSAEAD2022(ss.Method) {
		password = uuidToBase64Key(ss.Password)
	}

	userInfo := base64.StdEncoding.EncodeToString([]byte(ss.Method + ":" + password))
	return fmt.Sprintf("ss://%s@%s:%d#%s", userInfo, host, ib.Port, url.PathEscape(ib.Tag))
}

// isSSAEAD2022 returns true for SS2022 methods that require base64 keys.
func isSSAEAD2022(method string) bool {
	switch method {
	case "2022-blake3-aes-128-gcm", "2022-blake3-aes-256-gcm", "2022-blake3-chacha20-poly1305":
		return true
	}
	return false
}

// uuidToBase64Key parses a UUID string and returns its 16 bytes as base64.
func uuidToBase64Key(uuidStr string) string {
	// Parse UUID: remove hyphens, decode hex
	clean := ""
	for _, c := range uuidStr {
		if c != '-' {
			clean += string(c)
		}
	}
	if len(clean) != 32 {
		return uuidStr // fallback: return as-is if not a valid UUID
	}
	var raw [16]byte
	for i := 0; i < 16; i++ {
		fmt.Sscanf(clean[i*2:i*2+2], "%02x", &raw[i])
	}
	return base64.StdEncoding.EncodeToString(raw[:])
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/protocol/ -v -count=1 -run TestShadowsocks`

Expected: All 4 Shadowsocks tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/protocol/shadowsocks.go internal/protocol/shadowsocks_test.go
git commit -m "feat(protocol): Shadowsocks SS2022 implementation"
```

---

### Task 3: VLESS-REALITY Protocol Implementation

**Files:**
- Create: `internal/protocol/vless.go`
- Create: `internal/protocol/vless_test.go`

- [ ] **Step 1: Write VLESS tests**

Create `internal/protocol/vless_test.go`:

```go
package protocol

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/233boy/sing-box/internal/store"
)

func TestVLESSDefaultSettings(t *testing.T) {
	v := &VLESS{}
	settingsJSON, err := v.DefaultSettings(443)
	if err != nil {
		t.Fatalf("DefaultSettings error: %v", err)
	}

	var s vlessSettings
	if err := json.Unmarshal([]byte(settingsJSON), &s); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(s.UUID) != 36 {
		t.Errorf("UUID length = %d, want 36", len(s.UUID))
	}
	if s.Flow != "xtls-rprx-vision" {
		t.Errorf("flow = %q, want xtls-rprx-vision", s.Flow)
	}
	if s.Reality.PrivateKey == "" {
		t.Error("reality.private_key should not be empty")
	}
	if s.Reality.PublicKey == "" {
		t.Error("reality.public_key should not be empty")
	}
	if len(s.Reality.ShortID) != 8 {
		t.Errorf("reality.short_id length = %d, want 8", len(s.Reality.ShortID))
	}
	if s.Reality.Handshake.Server != "www.microsoft.com" {
		t.Errorf("handshake.server = %q, want www.microsoft.com", s.Reality.Handshake.Server)
	}
	if s.Reality.Handshake.ServerPort != 443 {
		t.Errorf("handshake.server_port = %d, want 443", s.Reality.Handshake.ServerPort)
	}
}

func TestVLESSBuildInbound(t *testing.T) {
	v := &VLESS{}
	settingsJSON, _ := v.DefaultSettings(443)
	ib := &store.Inbound{
		Tag:      "vless-443",
		Protocol: "vless",
		Port:     443,
		Settings: settingsJSON,
	}

	opt, err := v.BuildInbound(ib)
	if err != nil {
		t.Fatalf("BuildInbound error: %v", err)
	}
	if opt.Type != "vless" {
		t.Errorf("type = %q, want vless", opt.Type)
	}
	if opt.Tag != "vless-443" {
		t.Errorf("tag = %q, want vless-443", opt.Tag)
	}
}

func TestVLESSGenerateURL(t *testing.T) {
	v := &VLESS{}
	settingsJSON, _ := v.DefaultSettings(443)
	ib := &store.Inbound{
		Tag:      "vless-443",
		Port:     443,
		Settings: settingsJSON,
	}

	url := v.GenerateURL(ib, "1.2.3.4")
	if !strings.HasPrefix(url, "vless://") {
		t.Errorf("URL should start with vless://, got %q", url)
	}
	if !strings.Contains(url, "@1.2.3.4:443") {
		t.Errorf("URL should contain @1.2.3.4:443, got %q", url)
	}
	if !strings.Contains(url, "security=reality") {
		t.Errorf("URL should contain security=reality, got %q", url)
	}
	if !strings.Contains(url, "flow=xtls-rprx-vision") {
		t.Errorf("URL should contain flow=xtls-rprx-vision, got %q", url)
	}
	if !strings.Contains(url, "fp=chrome") {
		t.Errorf("URL should contain fp=chrome, got %q", url)
	}
	if !strings.Contains(url, "#vless-443") {
		t.Errorf("URL should end with #vless-443, got %q", url)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/protocol/ -v -count=1 -run TestVLESS`

Expected: Compilation error — `VLESS` type not defined.

- [ ] **Step 3: Implement vless.go**

Create `internal/protocol/vless.go`:

```go
package protocol

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/233boy/sing-box/internal/store"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
)

func init() {
	Register(&VLESS{})
}

type vlessRealityHandshake struct {
	Server     string `json:"server"`
	ServerPort uint16 `json:"server_port"`
}

type vlessReality struct {
	PrivateKey string                `json:"private_key"`
	PublicKey  string                `json:"public_key"`
	ShortID   string                `json:"short_id"`
	Handshake vlessRealityHandshake `json:"handshake"`
}

type vlessSettings struct {
	UUID    string       `json:"uuid"`
	Flow    string       `json:"flow"`
	Reality vlessReality `json:"reality"`
}

// VLESS implements Protocol for VLESS with REALITY.
type VLESS struct{}

func (v *VLESS) Name() string        { return "vless" }
func (v *VLESS) DisplayName() string  { return "VLESS-REALITY" }

func (v *VLESS) DefaultSettings(port uint16) (string, error) {
	privateKey, publicKey, err := generateX25519KeyPair()
	if err != nil {
		return "", fmt.Errorf("failed to generate X25519 keypair: %w", err)
	}

	settings := vlessSettings{
		UUID: GenerateUUID(),
		Flow: "xtls-rprx-vision",
		Reality: vlessReality{
			PrivateKey: privateKey,
			PublicKey:  publicKey,
			ShortID:   GenerateShortID(),
			Handshake: vlessRealityHandshake{
				Server:     "www.microsoft.com",
				ServerPort: 443,
			},
		},
	}
	b, err := json.Marshal(settings)
	return string(b), err
}

func (v *VLESS) BuildInbound(ib *store.Inbound) (option.Inbound, error) {
	var s vlessSettings
	if err := json.Unmarshal([]byte(ib.Settings), &s); err != nil {
		return option.Inbound{}, fmt.Errorf("invalid vless settings: %w", err)
	}

	return option.Inbound{
		Type: "vless",
		Tag:  ib.Tag,
		Options: &option.VLESSInboundOptions{
			ListenOptions: option.ListenOptions{
				ListenPort: ib.Port,
			},
			Users: []option.VLESSUser{
				{
					Name: "default",
					UUID: s.UUID,
					Flow: s.Flow,
				},
			},
			InboundTLSOptionsContainer: option.InboundTLSOptionsContainer{
				TLS: &option.InboundTLSOptions{
					Enabled:    true,
					ServerName: s.Reality.Handshake.Server,
					Reality: &option.InboundRealityOptions{
						Enabled: true,
						Handshake: option.InboundRealityHandshakeOptions{
							ServerOptions: option.ServerOptions{
								Server:     s.Reality.Handshake.Server,
								ServerPort: s.Reality.Handshake.ServerPort,
							},
						},
						PrivateKey: s.Reality.PrivateKey,
						ShortID:    badoption.Listable[string]{s.Reality.ShortID},
					},
				},
			},
		},
	}, nil
}

func (v *VLESS) GenerateURL(ib *store.Inbound, host string) string {
	var s vlessSettings
	if err := json.Unmarshal([]byte(ib.Settings), &s); err != nil {
		return ""
	}

	params := url.Values{}
	params.Set("type", "tcp")
	params.Set("security", "reality")
	params.Set("sni", s.Reality.Handshake.Server)
	params.Set("fp", "chrome")
	params.Set("pbk", s.Reality.PublicKey)
	params.Set("sid", s.Reality.ShortID)
	params.Set("flow", s.Flow)

	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		s.UUID, host, ib.Port, params.Encode(), url.PathEscape(ib.Tag))
}

// generateX25519KeyPair generates a Reality keypair using Go's crypto/ecdh.
func generateX25519KeyPair() (privateKeyStr, publicKeyStr string, err error) {
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	privateKeyStr = base64.RawURLEncoding.EncodeToString(key.Bytes())
	publicKeyStr = base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes())
	return privateKeyStr, publicKeyStr, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/protocol/ -v -count=1 -run TestVLESS`

Expected: All 3 VLESS tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/protocol/vless.go internal/protocol/vless_test.go
git commit -m "feat(protocol): VLESS-REALITY implementation with auto X25519 keypair"
```

---

### Task 4: Trojan Protocol Implementation

**Files:**
- Create: `internal/protocol/trojan.go`
- Create: `internal/protocol/trojan_test.go`

- [ ] **Step 1: Write Trojan tests**

Create `internal/protocol/trojan_test.go`:

```go
package protocol

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/233boy/sing-box/internal/store"
)

func TestTrojanDefaultSettings(t *testing.T) {
	tr := &Trojan{}
	settingsJSON, err := tr.DefaultSettings(443)
	if err != nil {
		t.Fatalf("DefaultSettings error: %v", err)
	}

	var s trojanSettings
	if err := json.Unmarshal([]byte(settingsJSON), &s); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(s.Password) != 36 {
		t.Errorf("password length = %d, want 36 (UUID)", len(s.Password))
	}
}

func TestTrojanBuildInbound(t *testing.T) {
	tr := &Trojan{}
	ib := &store.Inbound{
		Tag:      "trojan-443",
		Protocol: "trojan",
		Port:     443,
		Settings: `{"password":"550e8400-e29b-41d4-a716-446655440000"}`,
	}

	opt, err := tr.BuildInbound(ib)
	if err != nil {
		t.Fatalf("BuildInbound error: %v", err)
	}
	if opt.Type != "trojan" {
		t.Errorf("type = %q, want trojan", opt.Type)
	}
	if opt.Tag != "trojan-443" {
		t.Errorf("tag = %q, want trojan-443", opt.Tag)
	}
}

func TestTrojanBuildInboundInvalidJSON(t *testing.T) {
	tr := &Trojan{}
	ib := &store.Inbound{Settings: `{invalid}`}

	_, err := tr.BuildInbound(ib)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestTrojanGenerateURL(t *testing.T) {
	tr := &Trojan{}
	ib := &store.Inbound{
		Tag:      "trojan-443",
		Port:     443,
		Settings: `{"password":"550e8400-e29b-41d4-a716-446655440000"}`,
	}

	u := tr.GenerateURL(ib, "1.2.3.4")
	if !strings.HasPrefix(u, "trojan://") {
		t.Errorf("URL should start with trojan://, got %q", u)
	}
	if !strings.Contains(u, "550e8400-e29b-41d4-a716-446655440000@1.2.3.4:443") {
		t.Errorf("URL should contain password@host:port, got %q", u)
	}
	if !strings.HasSuffix(u, "#trojan-443") {
		t.Errorf("URL should end with #trojan-443, got %q", u)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/protocol/ -v -count=1 -run TestTrojan`

Expected: Compilation error — `Trojan` type not defined.

- [ ] **Step 3: Implement trojan.go**

Create `internal/protocol/trojan.go`:

```go
package protocol

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/233boy/sing-box/internal/store"
	"github.com/sagernet/sing-box/option"
)

func init() {
	Register(&Trojan{})
}

type trojanSettings struct {
	Password string `json:"password"`
}

// Trojan implements Protocol for Trojan (Phase 1: no TLS, suitable for reverse proxy setups).
type Trojan struct{}

func (tr *Trojan) Name() string        { return "trojan" }
func (tr *Trojan) DisplayName() string  { return "Trojan" }

func (tr *Trojan) DefaultSettings(port uint16) (string, error) {
	settings := trojanSettings{
		Password: GenerateUUID(),
	}
	b, err := json.Marshal(settings)
	return string(b), err
}

func (tr *Trojan) BuildInbound(ib *store.Inbound) (option.Inbound, error) {
	var s trojanSettings
	if err := json.Unmarshal([]byte(ib.Settings), &s); err != nil {
		return option.Inbound{}, fmt.Errorf("invalid trojan settings: %w", err)
	}

	return option.Inbound{
		Type: "trojan",
		Tag:  ib.Tag,
		Options: &option.TrojanInboundOptions{
			ListenOptions: option.ListenOptions{
				ListenPort: ib.Port,
			},
			Users: []option.TrojanUser{
				{
					Name:     "default",
					Password: s.Password,
				},
			},
		},
	}, nil
}

func (tr *Trojan) GenerateURL(ib *store.Inbound, host string) string {
	var s trojanSettings
	if err := json.Unmarshal([]byte(ib.Settings), &s); err != nil {
		return ""
	}

	return fmt.Sprintf("trojan://%s@%s:%d#%s",
		url.PathEscape(s.Password), host, ib.Port, url.PathEscape(ib.Tag))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/protocol/ -v -count=1 -run TestTrojan`

Expected: All 4 Trojan tests PASS.

- [ ] **Step 5: Run all protocol tests**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/protocol/ -v -count=1`

Expected: All protocol tests PASS (registry + shadowsocks + vless + trojan).

- [ ] **Step 6: Commit**

```bash
git add internal/protocol/trojan.go internal/protocol/trojan_test.go
git commit -m "feat(protocol): Trojan implementation"
```

---

### Task 5: Refactor Engine to Use Protocol Registry

**Files:**
- Modify: `internal/engine/build.go`
- Modify: `internal/engine/build_test.go`

- [ ] **Step 1: Update build_test.go for new protocol types**

Replace the entire content of `internal/engine/build_test.go`:

```go
package engine

import (
	"testing"

	"github.com/233boy/sing-box/internal/store"

	// Import protocol package to trigger init() registrations
	_ "github.com/233boy/sing-box/internal/protocol"
)

func TestBuildShadowsocksInbound(t *testing.T) {
	ib := &store.Inbound{
		ID:       1,
		Tag:      "ss-12345",
		Protocol: "shadowsocks",
		Port:     12345,
		Settings: `{"method":"2022-blake3-aes-128-gcm","password":"550e8400-e29b-41d4-a716-446655440000"}`,
	}

	opt, err := buildInbound(ib)
	if err != nil {
		t.Fatalf("buildInbound error: %v", err)
	}
	if opt.Type != "shadowsocks" {
		t.Errorf("type = %q, want shadowsocks", opt.Type)
	}
	if opt.Tag != "ss-12345" {
		t.Errorf("tag = %q, want ss-12345", opt.Tag)
	}
}

func TestBuildVLESSInbound(t *testing.T) {
	ib := &store.Inbound{
		Tag:      "vless-443",
		Protocol: "vless",
		Port:     443,
		Settings: `{"uuid":"550e8400-e29b-41d4-a716-446655440000","flow":"xtls-rprx-vision","reality":{"private_key":"testkey","public_key":"testpub","short_id":"abcd1234","handshake":{"server":"www.microsoft.com","server_port":443}}}`,
	}

	opt, err := buildInbound(ib)
	if err != nil {
		t.Fatalf("buildInbound error: %v", err)
	}
	if opt.Type != "vless" {
		t.Errorf("type = %q, want vless", opt.Type)
	}
}

func TestBuildTrojanInbound(t *testing.T) {
	ib := &store.Inbound{
		Tag:      "trojan-443",
		Protocol: "trojan",
		Port:     443,
		Settings: `{"password":"550e8400-e29b-41d4-a716-446655440000"}`,
	}

	opt, err := buildInbound(ib)
	if err != nil {
		t.Fatalf("buildInbound error: %v", err)
	}
	if opt.Type != "trojan" {
		t.Errorf("type = %q, want trojan", opt.Type)
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
}

func TestBuildOptionsWithInbounds(t *testing.T) {
	inbounds := []*store.Inbound{
		{Tag: "ss-1000", Protocol: "shadowsocks", Port: 1000, Settings: `{"method":"2022-blake3-aes-128-gcm","password":"550e8400-e29b-41d4-a716-446655440000"}`},
		{Tag: "trojan-2000", Protocol: "trojan", Port: 2000, Settings: `{"password":"550e8400-e29b-41d4-a716-446655440000"}`},
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

- [ ] **Step 2: Rewrite build.go to use protocol registry**

Replace the entire content of `internal/engine/build.go`:

```go
package engine

import (
	"fmt"

	"github.com/233boy/sing-box/internal/protocol"
	"github.com/233boy/sing-box/internal/store"
	"github.com/sagernet/sing-box/option"
)

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

// buildInbound delegates to the registered protocol implementation.
func buildInbound(ib *store.Inbound) (option.Inbound, error) {
	p := protocol.Get(ib.Protocol)
	if p == nil {
		return option.Inbound{}, fmt.Errorf("unsupported protocol: %s", ib.Protocol)
	}
	return p.BuildInbound(ib)
}
```

- [ ] **Step 3: Run engine tests**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/engine/ -v -count=1`

Expected: All 6 tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/engine/build.go internal/engine/build_test.go
git commit -m "refactor(engine): use protocol registry instead of switch-case"
```

---

### Task 6: Refactor API Inbound Handlers

**Files:**
- Modify: `internal/api/inbound.go`
- Modify: `internal/api/api_test.go`

- [ ] **Step 1: Update api_test.go for new behavior**

In `internal/api/api_test.go`, update `TestCreateInbound` to test auto-generated settings and the new tag format. Also add a test for creating without settings:

Add these tests to the existing file:

```go
func TestCreateInboundAutoSettings(t *testing.T) {
	srv, _, eng := setupTestServer(t)
	body := `{"protocol":"shadowsocks","port":8388}`
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
	if resp["tag"] != "shadowsocks-8388" {
		t.Errorf("tag = %v, want 'shadowsocks-8388'", resp["tag"])
	}
	// Settings should be auto-generated
	settings, ok := resp["settings"].(string)
	if !ok || settings == "" || settings == "{}" {
		t.Errorf("settings should be auto-generated, got %v", resp["settings"])
	}
	if eng.reloaded != 1 {
		t.Errorf("engine.Reload() called %d times, want 1", eng.reloaded)
	}
}

func TestCreateInboundVLESS(t *testing.T) {
	srv, _, _ := setupTestServer(t)
	body := `{"protocol":"vless","port":443}`
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
	if resp["tag"] != "vless-443" {
		t.Errorf("tag = %v, want 'vless-443'", resp["tag"])
	}
}

func TestCreateInboundUnsupportedProtocol(t *testing.T) {
	srv, _, _ := setupTestServer(t)
	body := `{"protocol":"unknown","port":1234}`
	req := httptest.NewRequest("POST", "/api/inbounds", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
```

- [ ] **Step 2: Update the existing TestCreateInbound test**

The existing test sends explicit settings — update its expected tag from `"ss-12345"` to `"shadowsocks-12345"`:

In the existing `TestCreateInbound`, change:
```go
if resp["tag"] != "ss-12345" {
```
to:
```go
if resp["tag"] != "shadowsocks-12345" {
```

- [ ] **Step 3: Rewrite handleCreateInbound in inbound.go**

In `internal/api/inbound.go`, update the import block to add the protocol package, then replace `handleCreateInbound`:

Add to imports:
```go
"github.com/233boy/sing-box/internal/protocol"
// Import all protocol implementations
_ "github.com/233boy/sing-box/internal/protocol"
```

Note: The blank import is only needed if the api package doesn't already transitively import protocol. Since protocol files use `init()` for registration, at least one import is needed somewhere in the import chain.

Replace `handleCreateInbound`:

```go
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

	// Validate protocol is supported
	p := protocol.Get(req.Protocol)
	if p == nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unsupported protocol: %s", req.Protocol))
		return
	}

	// Generate or use provided settings
	var settingsStr string
	if req.Settings == nil || string(req.Settings) == "{}" || string(req.Settings) == "null" {
		generated, err := p.DefaultSettings(req.Port)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to generate settings: "+err.Error())
			return
		}
		settingsStr = generated
	} else {
		settingsStr = string(req.Settings)
	}

	tag := fmt.Sprintf("%s-%d", req.Protocol, req.Port)

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

	s.engine.Reload()
	writeJSON(w, http.StatusCreated, ib)
}
```

- [ ] **Step 4: Run API tests**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/api/ -v -count=1`

Expected: All tests PASS (including the new ones).

- [ ] **Step 5: Commit**

```bash
git add internal/api/inbound.go internal/api/api_test.go
git commit -m "refactor(api): use protocol registry for DefaultSettings and tag generation"
```

---

### Task 7: Store Migration + Traffic Queries

**Files:**
- Modify: `internal/store/store.go`
- Create: `internal/store/traffic.go`
- Create: `internal/store/traffic_test.go`

- [ ] **Step 1: Write traffic store tests**

Create `internal/store/traffic_test.go`:

```go
package store

import (
	"testing"
)

func TestInsertTrafficLog(t *testing.T) {
	s := mustOpenTestStore(t)

	err := s.InsertTrafficLog("ss-8388", 1024, 2048)
	if err != nil {
		t.Fatalf("InsertTrafficLog error: %v", err)
	}
}

func TestGetTrafficSummary(t *testing.T) {
	s := mustOpenTestStore(t)

	s.InsertTrafficLog("ss-8388", 1000, 2000)
	s.InsertTrafficLog("ss-8388", 500, 1000)
	s.InsertTrafficLog("vless-443", 300, 600)

	summaries, err := s.GetTrafficSummary()
	if err != nil {
		t.Fatalf("GetTrafficSummary error: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}

	// Find ss-8388
	var found bool
	for _, s := range summaries {
		if s.Tag == "ss-8388" {
			found = true
			if s.Upload != 1500 {
				t.Errorf("ss-8388 upload = %d, want 1500", s.Upload)
			}
			if s.Download != 3000 {
				t.Errorf("ss-8388 download = %d, want 3000", s.Download)
			}
		}
	}
	if !found {
		t.Error("ss-8388 not found in summaries")
	}
}

func TestGetTrafficByTag(t *testing.T) {
	s := mustOpenTestStore(t)

	s.InsertTrafficLog("ss-8388", 1000, 2000)
	s.InsertTrafficLog("ss-8388", 500, 1000)

	logs, err := s.GetTrafficByTag("ss-8388")
	if err != nil {
		t.Fatalf("GetTrafficByTag error: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logs))
	}
}

func TestGetTrafficByTagEmpty(t *testing.T) {
	s := mustOpenTestStore(t)

	logs, err := s.GetTrafficByTag("nonexistent")
	if err != nil {
		t.Fatalf("GetTrafficByTag error: %v", err)
	}
	if len(logs) != 0 {
		t.Fatalf("expected 0 logs, got %d", len(logs))
	}
}
```

- [ ] **Step 2: Add migration 2 to store.go**

In `internal/store/store.go`, add the traffic_logs migration to the `migrations` slice:

```go
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
	// Migration 2: traffic_logs table
	`CREATE TABLE IF NOT EXISTS traffic_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		inbound_tag TEXT NOT NULL,
		upload INTEGER NOT NULL,
		download INTEGER NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
}
```

- [ ] **Step 3: Implement traffic.go**

Create `internal/store/traffic.go`:

```go
package store

import (
	"fmt"
	"time"
)

// TrafficLog represents a single traffic measurement.
type TrafficLog struct {
	ID         int64     `json:"id"`
	InboundTag string    `json:"inbound_tag"`
	Upload     int64     `json:"upload"`
	Download   int64     `json:"download"`
	Timestamp  time.Time `json:"timestamp"`
}

// TrafficSummary holds aggregated traffic for one inbound.
type TrafficSummary struct {
	Tag      string `json:"tag"`
	Upload   int64  `json:"upload"`
	Download int64  `json:"download"`
}

// InsertTrafficLog records a traffic measurement.
func (s *Store) InsertTrafficLog(inboundTag string, upload, download int64) error {
	_, err := s.db.Exec(
		`INSERT INTO traffic_logs (inbound_tag, upload, download) VALUES (?, ?, ?)`,
		inboundTag, upload, download,
	)
	if err != nil {
		return fmt.Errorf("failed to insert traffic log: %w", err)
	}
	return nil
}

// GetTrafficSummary returns total upload/download per inbound.
func (s *Store) GetTrafficSummary() ([]TrafficSummary, error) {
	rows, err := s.db.Query(
		`SELECT inbound_tag, SUM(upload), SUM(download) FROM traffic_logs GROUP BY inbound_tag ORDER BY inbound_tag`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query traffic summary: %w", err)
	}
	defer rows.Close()

	var summaries []TrafficSummary
	for rows.Next() {
		var ts TrafficSummary
		if err := rows.Scan(&ts.Tag, &ts.Upload, &ts.Download); err != nil {
			return nil, fmt.Errorf("failed to scan traffic summary: %w", err)
		}
		summaries = append(summaries, ts)
	}
	return summaries, rows.Err()
}

// GetTrafficByTag returns traffic log entries for a specific inbound.
func (s *Store) GetTrafficByTag(tag string) ([]TrafficLog, error) {
	rows, err := s.db.Query(
		`SELECT id, inbound_tag, upload, download, timestamp FROM traffic_logs WHERE inbound_tag = ? ORDER BY timestamp DESC`,
		tag,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query traffic by tag: %w", err)
	}
	defer rows.Close()

	var logs []TrafficLog
	for rows.Next() {
		var tl TrafficLog
		if err := rows.Scan(&tl.ID, &tl.InboundTag, &tl.Upload, &tl.Download, &tl.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan traffic log: %w", err)
		}
		logs = append(logs, tl)
	}
	return logs, rows.Err()
}
```

- [ ] **Step 4: Run traffic store tests**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/store/ -v -count=1 -run TestTraffic`

Expected: All 4 traffic tests PASS.

- [ ] **Step 5: Run all store tests**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/store/ -v -count=1`

Expected: All store tests PASS (existing + new).

- [ ] **Step 6: Commit**

```bash
git add internal/store/store.go internal/store/traffic.go internal/store/traffic_test.go
git commit -m "feat(store): traffic_logs migration and traffic query methods"
```

---

### Task 8: Stats Tracker + Engine Integration

**Files:**
- Create: `internal/stats/tracker.go`
- Create: `internal/stats/tracker_test.go`
- Create: `internal/stats/collector.go`
- Create: `internal/stats/collector_test.go`
- Modify: `internal/engine/engine.go`

- [ ] **Step 1: Write tracker tests**

Create `internal/stats/tracker_test.go`:

```go
package stats

import (
	"testing"
)

func TestTrackerGetOrCreate(t *testing.T) {
	tr := NewTracker()

	s1 := tr.getOrCreate("ss-8388")
	s1.Upload.Add(1000)
	s1.Download.Add(2000)

	s2 := tr.getOrCreate("ss-8388")
	if s2.Upload.Load() != 1000 {
		t.Errorf("upload = %d, want 1000", s2.Upload.Load())
	}
	if s2.Download.Load() != 2000 {
		t.Errorf("download = %d, want 2000", s2.Download.Load())
	}
}

func TestTrackerSnapshot(t *testing.T) {
	tr := NewTracker()

	s1 := tr.getOrCreate("ss-8388")
	s1.Upload.Add(1000)
	s1.Download.Add(2000)

	s2 := tr.getOrCreate("vless-443")
	s2.Upload.Add(500)
	s2.Download.Add(1000)

	snap := tr.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("snapshot has %d entries, want 2", len(snap))
	}

	if snap["ss-8388"].Upload != 1000 || snap["ss-8388"].Download != 2000 {
		t.Errorf("ss-8388 = %+v, want {1000, 2000}", snap["ss-8388"])
	}
	if snap["vless-443"].Upload != 500 || snap["vless-443"].Download != 1000 {
		t.Errorf("vless-443 = %+v, want {500, 1000}", snap["vless-443"])
	}
}
```

- [ ] **Step 2: Implement tracker.go**

Create `internal/stats/tracker.go`:

```go
package stats

import (
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/sagernet/sing-box/adapter"
	N "github.com/sagernet/sing/common/network"
)

// TrafficCounter holds upload/download byte counts.
type TrafficCounter struct {
	Upload   int64
	Download int64
}

// inboundStats tracks per-inbound traffic with atomic counters.
type inboundStats struct {
	Upload   atomic.Int64
	Download atomic.Int64
}

// Tracker implements adapter.ConnectionTracker to count per-inbound traffic.
type Tracker struct {
	mu    sync.RWMutex
	stats map[string]*inboundStats
}

// NewTracker creates a new Tracker.
func NewTracker() *Tracker {
	return &Tracker{
		stats: make(map[string]*inboundStats),
	}
}

func (t *Tracker) getOrCreate(tag string) *inboundStats {
	t.mu.RLock()
	s, ok := t.stats[tag]
	t.mu.RUnlock()
	if ok {
		return s
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	// Double-check after write lock
	if s, ok := t.stats[tag]; ok {
		return s
	}
	s = &inboundStats{}
	t.stats[tag] = s
	return s
}

// Snapshot returns a point-in-time copy of all traffic counters.
func (t *Tracker) Snapshot() map[string]TrafficCounter {
	t.mu.RLock()
	defer t.mu.RUnlock()

	snap := make(map[string]TrafficCounter, len(t.stats))
	for tag, s := range t.stats {
		snap[tag] = TrafficCounter{
			Upload:   s.Upload.Load(),
			Download: s.Download.Load(),
		}
	}
	return snap
}

// RoutedConnection wraps a TCP connection with upload/download counting.
func (t *Tracker) RoutedConnection(ctx context.Context, conn net.Conn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) net.Conn {
	tag := metadata.Inbound
	if tag == "" {
		return conn
	}
	s := t.getOrCreate(tag)
	return &countConn{Conn: conn, upload: &s.Upload, download: &s.Download}
}

// RoutedPacketConnection wraps a UDP connection with upload/download counting.
func (t *Tracker) RoutedPacketConnection(ctx context.Context, conn N.PacketConn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) N.PacketConn {
	// Packet-level counting is more complex; for Phase 1 we skip UDP stats.
	// TCP covers the majority of proxy traffic.
	return conn
}

// countConn wraps net.Conn to count bytes read (download) and written (upload).
type countConn struct {
	net.Conn
	upload   *atomic.Int64
	download *atomic.Int64
}

func (c *countConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if n > 0 {
		c.download.Add(int64(n))
	}
	return
}

func (c *countConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	if n > 0 {
		c.upload.Add(int64(n))
	}
	return
}

func (c *countConn) WriteTo(w io.Writer) (n int64, err error) {
	if wt, ok := c.Conn.(io.WriterTo); ok {
		n, err = wt.WriteTo(w)
	} else {
		n, err = io.Copy(w, c.Conn)
	}
	if n > 0 {
		c.download.Add(n)
	}
	return
}
```

- [ ] **Step 3: Write collector tests**

Create `internal/stats/collector_test.go`:

```go
package stats

import (
	"testing"

	"github.com/233boy/sing-box/internal/store"
)

func TestCollectorFlush(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	tracker := NewTracker()
	s := tracker.getOrCreate("ss-8388")
	s.Upload.Add(1000)
	s.Download.Add(2000)

	c := NewCollector(tracker, st)
	if err := c.Flush(); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	// Verify data was written
	summaries, err := st.GetTrafficSummary()
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].Upload != 1000 {
		t.Errorf("upload = %d, want 1000", summaries[0].Upload)
	}
}

func TestCollectorFlushDelta(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	tracker := NewTracker()
	s := tracker.getOrCreate("ss-8388")
	s.Upload.Add(1000)
	s.Download.Add(2000)

	c := NewCollector(tracker, st)

	// First flush
	c.Flush()

	// Add more traffic
	s.Upload.Add(500)
	s.Download.Add(1000)

	// Second flush should only write the delta
	c.Flush()

	summaries, _ := st.GetTrafficSummary()
	if summaries[0].Upload != 1500 {
		t.Errorf("total upload = %d, want 1500 (1000 + 500)", summaries[0].Upload)
	}
	if summaries[0].Download != 3000 {
		t.Errorf("total download = %d, want 3000 (2000 + 1000)", summaries[0].Download)
	}
}
```

- [ ] **Step 4: Implement collector.go**

Create `internal/stats/collector.go`:

```go
package stats

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/233boy/sing-box/internal/store"
)

// Collector periodically flushes tracker deltas to the database.
type Collector struct {
	tracker  *Tracker
	store    *store.Store
	mu       sync.Mutex
	lastSnap map[string]TrafficCounter
}

// NewCollector creates a Collector.
func NewCollector(tracker *Tracker, st *store.Store) *Collector {
	return &Collector{
		tracker:  tracker,
		store:    st,
		lastSnap: make(map[string]TrafficCounter),
	}
}

// Flush computes deltas since the last flush and writes them to the database.
func (c *Collector) Flush() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	current := c.tracker.Snapshot()

	for tag, cur := range current {
		last := c.lastSnap[tag]

		deltaUp := cur.Upload - last.Upload
		deltaDown := cur.Download - last.Download

		// Handle counter reset (e.g., after engine reload)
		if deltaUp < 0 {
			deltaUp = cur.Upload
		}
		if deltaDown < 0 {
			deltaDown = cur.Download
		}

		if deltaUp == 0 && deltaDown == 0 {
			continue
		}

		if err := c.store.InsertTrafficLog(tag, deltaUp, deltaDown); err != nil {
			slog.Error("failed to insert traffic log", "tag", tag, "error", err)
			continue
		}
	}

	c.lastSnap = current
	return nil
}

// Run starts the periodic flush loop. Blocks until ctx is cancelled.
func (c *Collector) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Final flush before exit
			c.Flush()
			return
		case <-ticker.C:
			c.Flush()
		}
	}
}
```

- [ ] **Step 5: Run stats tests**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/stats/ -v -count=1`

Expected: All 4 stats tests PASS.

- [ ] **Step 6: Update engine.go — add tracker field and registration**

In `internal/engine/engine.go`, make these changes:

Add import:
```go
"github.com/233boy/sing-box/internal/stats"
```

Add `tracker` field to Engine struct:
```go
type Engine struct {
	store     *store.Store
	mu        sync.Mutex
	instance  *box.Box
	running   bool
	startedAt time.Time
	tracker   *stats.Tracker
}
```

Update `New()`:
```go
func New(s *store.Store) *Engine {
	return &Engine{store: s, tracker: stats.NewTracker()}
}
```

In `Start()`, after `instance.Start()` succeeds, add:
```go
e.instance.Router().AppendTracker(e.tracker)
```

In `Reload()`, after the new `instance.Start()` succeeds, add:
```go
e.instance.Router().AppendTracker(e.tracker)
```

Add `Tracker()` method:
```go
// Tracker returns the traffic tracker for use by the stats collector.
func (e *Engine) Tracker() *stats.Tracker {
	return e.tracker
}
```

- [ ] **Step 7: Verify engine still compiles**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go build ./internal/engine/`

Expected: Compiles without errors.

- [ ] **Step 8: Commit**

```bash
git add internal/stats/ internal/engine/engine.go
git commit -m "feat(stats): traffic tracker with per-inbound counting and periodic DB flush"
```

---

### Task 9: Stats API + Subscription Endpoint

**Files:**
- Create: `internal/api/stats.go`
- Create: `internal/api/subscription.go`
- Modify: `internal/api/server.go`
- Modify: `internal/api/api_test.go`

- [ ] **Step 1: Add stats and subscription tests to api_test.go**

Append to `internal/api/api_test.go`:

```go
func TestGetStats(t *testing.T) {
	srv, st, _ := setupTestServer(t)

	st.InsertTrafficLog("ss-8388", 1000, 2000)
	st.InsertTrafficLog("ss-8388", 500, 1000)

	req := httptest.NewRequest("GET", "/api/stats", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp []map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(resp))
	}
}

func TestGetStatsByTag(t *testing.T) {
	srv, st, _ := setupTestServer(t)

	st.InsertTrafficLog("ss-8388", 1000, 2000)

	req := httptest.NewRequest("GET", "/api/stats/ss-8388", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSubscriptionValidToken(t *testing.T) {
	srv, st, _ := setupTestServer(t)

	st.CreateInbound(&store.Inbound{
		Tag: "shadowsocks-8388", Protocol: "shadowsocks", Port: 8388,
		Settings: `{"method":"2022-blake3-aes-128-gcm","password":"550e8400-e29b-41d4-a716-446655440000"}`,
	})

	req := httptest.NewRequest("GET", "/sub/test-sub-token", nil)
	req.Host = "1.2.3.4:9090"
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("content-type = %q, want text/plain", ct)
	}
	// Body should be non-empty base64
	if w.Body.Len() == 0 {
		t.Error("response body should not be empty")
	}
}

func TestSubscriptionInvalidToken(t *testing.T) {
	srv, _, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/sub/wrong-token", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
```

- [ ] **Step 2: Update setupTestServer to pass subToken**

In `internal/api/api_test.go`, update `setupTestServer`:

```go
func setupTestServer(t *testing.T) (*Server, *store.Store, *mockEngine) {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	eng := &mockEngine{}
	srv := NewServer(eng, st, nil, "127.0.0.1:0", "test-token", "test-sub-token")
	return srv, st, eng
}
```

- [ ] **Step 3: Implement stats.go**

Create `internal/api/stats.go`:

```go
package api

import (
	"net/http"

	"github.com/233boy/sing-box/internal/store"
	"github.com/go-chi/chi/v5"
)

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	summaries, err := s.store.GetTrafficSummary()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if summaries == nil {
		summaries = []store.TrafficSummary{}
	}
	writeJSON(w, http.StatusOK, summaries)
}

func (s *Server) handleGetStatsByTag(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")

	logs, err := s.store.GetTrafficByTag(tag)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if logs == nil {
		logs = []store.TrafficLog{}
	}
	writeJSON(w, http.StatusOK, logs)
}
```

- [ ] **Step 4: Implement subscription.go**

Create `internal/api/subscription.go`:

```go
package api

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/233boy/sing-box/internal/protocol"
	"github.com/go-chi/chi/v5"

	// Ensure all protocols are registered
	_ "github.com/233boy/sing-box/internal/protocol"
)

func (s *Server) handleSubscription(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token != s.subToken {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	inbounds, err := s.store.ListInbounds()
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	host := r.Host
	// Strip port for share URLs if it's the API port
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	var urls []string
	for _, ib := range inbounds {
		p := protocol.Get(ib.Protocol)
		if p == nil {
			continue
		}
		u := p.GenerateURL(ib, host)
		if u != "" {
			urls = append(urls, u)
		}
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(strings.Join(urls, "\n")))

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(encoded))
}
```

- [ ] **Step 5: Update server.go — add subToken field and route groups**

Rewrite `internal/api/server.go`:

```go
package api

import (
	"context"
	"net"
	"net/http"

	"github.com/233boy/sing-box/internal/store"
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
	engine   EngineController
	store    *store.Store
	router   chi.Router
	httpSrv  *http.Server
	token    string
	subToken string
}

// NewServer creates an API server wired to the given engine and store.
// svc may be nil if service management is unavailable on this platform.
func NewServer(engine EngineController, st *store.Store, svc any, listenAddr, token, subToken string) *Server {
	s := &Server{
		engine:   engine,
		store:    st,
		token:    token,
		subToken: subToken,
	}

	r := chi.NewRouter()

	// Public routes (no auth)
	r.Get("/sub/{token}", s.handleSubscription)

	// Authenticated API routes
	r.Group(func(r chi.Router) {
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

		// Traffic stats
		r.Get("/api/stats", s.handleGetStats)
		r.Get("/api/stats/{tag}", s.handleGetStatsByTag)
	})

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

- [ ] **Step 6: Run API tests**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/api/ -v -count=1`

Expected: All tests PASS (existing + new stats + subscription tests).

- [ ] **Step 7: Commit**

```bash
git add internal/api/stats.go internal/api/subscription.go internal/api/server.go internal/api/api_test.go
git commit -m "feat(api): stats endpoints and subscription with Base64 share URLs"
```

---

### Task 10: Main Entry Point Wiring

**Files:**
- Modify: `cmd/sing-box/main.go`

- [ ] **Step 1: Update main.go to wire collector, sub_token, and protocol imports**

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

	"github.com/233boy/sing-box/internal/api"
	"github.com/233boy/sing-box/internal/engine"
	"github.com/233boy/sing-box/internal/platform"
	"github.com/233boy/sing-box/internal/service"
	"github.com/233boy/sing-box/internal/stats"
	"github.com/233boy/sing-box/internal/store"
	"github.com/spf13/cobra"

	// Register all protocol implementations
	_ "github.com/233boy/sing-box/internal/protocol"
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
	apiToken, err := ensureToken(st, "api_token")
	if err != nil {
		return fmt.Errorf("无法初始化 API token: %w", err)
	}

	// Ensure subscription token exists
	subToken, err := ensureToken(st, "sub_token")
	if err != nil {
		return fmt.Errorf("无法初始化订阅 token: %w", err)
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

	// Start traffic collector
	collector := stats.NewCollector(eng.Tracker(), st)
	collectorCtx, collectorCancel := context.WithCancel(context.Background())
	go collector.Run(collectorCtx, 60*time.Second)

	// Create and start API server
	srv := api.NewServer(eng, st, svcMgr, listenAddr, apiToken, subToken)
	if err := srv.Start(); err != nil {
		collectorCancel()
		return fmt.Errorf("无法启动 API 服务: %w", err)
	}

	slog.Info("sing-box 面板已启动",
		"listen", listenAddr,
		"data_dir", dataDir,
		"init_system", plat.InitSystem,
	)
	fmt.Printf("\nAPI Token: %s\n", apiToken)
	fmt.Printf("API 地址: http://%s\n", listenAddr)
	fmt.Printf("订阅地址: http://%s/sub/%s\n\n", listenAddr, subToken)

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
			collectorCancel()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			srv.Shutdown(shutdownCtx)
			eng.Stop()
			slog.Info("已关闭")
			return nil
		}
	}
}

func ensureToken(st *store.Store, key string) (string, error) {
	token, err := st.GetSetting(key)
	if err != nil {
		return "", err
	}
	if token == "" {
		token = generateToken()
		if err := st.SetSetting(key, token); err != nil {
			return "", err
		}
	}
	return token, nil
}

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go build -o sing-box-m2 ./cmd/sing-box/`

Expected: Binary `sing-box-m2` built successfully.

- [ ] **Step 3: Verify help output**

Run: `./sing-box-m2 serve --help`

Expected: Shows `--listen` and `--data-dir` flags.

- [ ] **Step 4: Commit**

```bash
git add cmd/sing-box/main.go
git commit -m "feat: M2 — wire protocol registry, stats collector, and subscription endpoint"
```

---

### Task 11: Full Test Suite + Verification

- [ ] **Step 1: Run full test suite**

```bash
cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/... -v -count=1
```

Expected: All tests across protocol, store, engine, service, api, and stats packages PASS.

- [ ] **Step 2: Run go vet**

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

- [ ] **Step 4: Smoke test (if on Linux or with local sing-box binary)**

```bash
mkdir -p /tmp/sing-box-m2-test
./sing-box-m2 serve --data-dir /tmp/sing-box-m2-test --listen 127.0.0.1:19090 &
SERVE_PID=$!
sleep 2

# Get tokens from DB
TOKEN=$(sqlite3 /tmp/sing-box-m2-test/panel.db "SELECT value FROM settings WHERE key='api_token'")
SUB_TOKEN=$(sqlite3 /tmp/sing-box-m2-test/panel.db "SELECT value FROM settings WHERE key='sub_token'")

# Create Shadowsocks (auto settings)
curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"protocol":"shadowsocks","port":18388}' \
  http://127.0.0.1:19090/api/inbounds | python3 -m json.tool

# Create VLESS-REALITY (auto settings)
curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"protocol":"vless","port":18443}' \
  http://127.0.0.1:19090/api/inbounds | python3 -m json.tool

# Create Trojan (auto settings)
curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"protocol":"trojan","port":18444}' \
  http://127.0.0.1:19090/api/inbounds | python3 -m json.tool

# List all inbounds
curl -s -H "Authorization: Bearer $TOKEN" http://127.0.0.1:19090/api/inbounds | python3 -m json.tool

# Get subscription
curl -s http://127.0.0.1:19090/sub/$SUB_TOKEN | base64 -d

# Get stats (should be empty initially)
curl -s -H "Authorization: Bearer $TOKEN" http://127.0.0.1:19090/api/stats | python3 -m json.tool

# Cleanup
kill $SERVE_PID
rm -rf /tmp/sing-box-m2-test
```

Expected:
- Each create returns 201 with auto-generated settings and tag like `shadowsocks-18388`, `vless-18443`, `trojan-18444`
- List returns 3 inbounds
- Subscription returns Base64 decoded to 3 share URLs (ss://, vless://, trojan://)
- Stats returns empty array (no traffic yet)

- [ ] **Step 5: Fix any issues found and commit**

If any fixes needed:

```bash
git add -A
git commit -m "fix: M2 test and verification fixes"
```

- [ ] **Step 6: Clean up build artifact**

```bash
rm -f sing-box-m2
echo "sing-box-m2" >> .gitignore
git add .gitignore
git commit -m "chore: add M2 build artifact to gitignore"
```

---

## Summary

| Package | New/Modified | Key Changes |
|---------|-------------|-------------|
| `protocol` | **New** | Protocol interface, registry, SS/VLESS/Trojan implementations |
| `stats` | **New** | ConnectionTracker, per-inbound counters, periodic DB flush |
| `store` | Modified | Migration 2 (traffic_logs), traffic query methods |
| `engine` | Modified | Uses protocol registry, registers tracker on box.Router |
| `api` | Modified | Route groups, stats endpoints, subscription endpoint, DefaultSettings |
| `main` | Modified | Wires collector, sub_token, protocol imports |

## API Endpoints (M2 additions)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/stats` | Bearer | All inbound traffic summaries |
| GET | `/api/stats/{tag}` | Bearer | Single inbound traffic history |
| GET | `/sub/{token}` | URL token | Base64 subscription (all inbounds) |

## Protocols

| Protocol | Tag Format | Share URL Prefix | Auto-Generated Fields |
|----------|-----------|-----------------|----------------------|
| Shadowsocks | `shadowsocks-{port}` | `ss://` | method, password (UUID→base64) |
| VLESS-REALITY | `vless-{port}` | `vless://` | uuid, X25519 keypair, short_id |
| Trojan | `trojan-{port}` | `trojan://` | password (UUID) |
