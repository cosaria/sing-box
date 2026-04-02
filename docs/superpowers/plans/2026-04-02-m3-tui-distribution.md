# M3: TUI + Distribution Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Bubbletea TUI, QR code display, install.sh, self-update, and goreleaser to deliver v0.1.

**Architecture:** TUI is a Bubbletea app running in the same process as the API server. It communicates via localhost HTTP through an API client. Updater downloads binaries from GitHub Releases. install.sh is a standalone bash script.

**Tech Stack:** Go 1.26, bubbletea v1.3.10, lipgloss v1.1.0, bubbles v1.0.0, go-qrcode, cobra

**Module path:** `github.com/233boy/sing-box`

---

## File Structure

```
internal/
├── tui/
│   ├── client.go           # HTTP API client
│   ├── client_test.go      # Client tests (httptest)
│   ├── qr.go               # QR terminal renderer
│   ├── qr_test.go          # QR tests
│   ├── style.go            # Lipgloss style constants
│   ├── tui.go              # App model + state machine + Run()
│   ├── menu.go             # Main menu view
│   ├── add.go              # Add inbound wizard
│   ├── list.go             # Inbound list + detail + QR display
│   ├── edit.go             # Edit inbound view
│   ├── status.go           # Engine status + subscription URL
│   └── stats.go            # Traffic stats table
├── updater/
│   ├── updater.go          # Check + download + replace binary
│   └── updater_test.go     # Updater tests
cmd/sing-box/main.go        # (modify) TUI mode + update/service/version commands
install.sh                   # Installation script
.goreleaser.yml              # Build configuration
```

## Dependency Graph

```
Group A (no deps):          client.go, qr.go, style.go
Group B (needs A):          tui.go (core model)
Group C (needs B):          menu.go, add.go, list.go, edit.go, status.go, stats.go
Group D (no deps):          updater.go
Group E (needs C+D):        main.go
Group F (no deps):          install.sh, .goreleaser.yml
Group G (final):            verification
```

---

### Task 1: Add Dependencies

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add TUI dependencies**

```bash
cd /Users/admin/Codes/ProxyCode/sing-box
go get github.com/charmbracelet/bubbletea@v1.3.10
go get github.com/charmbracelet/lipgloss@v1.1.0
go get github.com/charmbracelet/bubbles@v1.0.0
go get github.com/skip2/go-qrcode@latest
```

- [ ] **Step 2: Create directories**

```bash
mkdir -p internal/tui internal/updater
```

- [ ] **Step 3: Verify dependencies**

```bash
grep -E "bubbletea|lipgloss|bubbles|qrcode" go.mod
```

Expected: All four packages listed.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add M3 dependencies — bubbletea, lipgloss, bubbles, go-qrcode"
```

---

### Task 2: API Client

**Files:**
- Create: `internal/tui/client.go`
- Create: `internal/tui/client_test.go`

- [ ] **Step 1: Write client tests**

Create `internal/tui/client_test.go`:

```go
package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/status" {
			t.Errorf("path = %q, want /api/status", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing auth header")
		}
		json.NewEncoder(w).Encode(map[string]any{"engine": "running", "inbounds": 2})
	}))
	defer ts.Close()

	c := NewClient(ts.URL, "test-token")
	resp, err := c.Status()
	if err != nil {
		t.Fatalf("Status error: %v", err)
	}
	if resp.Engine != "running" {
		t.Errorf("engine = %q, want running", resp.Engine)
	}
	if resp.Inbounds != 2 {
		t.Errorf("inbounds = %d, want 2", resp.Inbounds)
	}
}

func TestClientListInbounds(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "tag": "ss-8388", "protocol": "shadowsocks", "port": 8388, "settings": "{}"},
		})
	}))
	defer ts.Close()

	c := NewClient(ts.URL, "test-token")
	list, err := c.ListInbounds()
	if err != nil {
		t.Fatalf("ListInbounds error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 inbound, got %d", len(list))
	}
	if list[0].Tag != "ss-8388" {
		t.Errorf("tag = %q, want ss-8388", list[0].Tag)
	}
}

func TestClientCreateInbound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id": 1, "tag": "shadowsocks-8388", "protocol": "shadowsocks", "port": 8388, "settings": `{"method":"2022-blake3-aes-128-gcm","password":"test"}`,
		})
	}))
	defer ts.Close()

	c := NewClient(ts.URL, "test-token")
	ib, err := c.CreateInbound("shadowsocks", 8388)
	if err != nil {
		t.Fatalf("CreateInbound error: %v", err)
	}
	if ib.Tag != "shadowsocks-8388" {
		t.Errorf("tag = %q, want shadowsocks-8388", ib.Tag)
	}
}

func TestClientDeleteInbound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := NewClient(ts.URL, "test-token")
	err := c.DeleteInbound(1)
	if err != nil {
		t.Fatalf("DeleteInbound error: %v", err)
	}
}

func TestClientGetStats(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{"tag": "ss-8388", "upload": 1024, "download": 2048},
		})
	}))
	defer ts.Close()

	c := NewClient(ts.URL, "test-token")
	stats, err := c.GetStats()
	if err != nil {
		t.Fatalf("GetStats error: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat, got %d", len(stats))
	}
	if stats[0].Upload != 1024 {
		t.Errorf("upload = %d, want 1024", stats[0].Upload)
	}
}
```

- [ ] **Step 2: Implement client.go**

Create `internal/tui/client.go`:

```go
package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type StatusResp struct {
	Engine   string `json:"engine"`
	Inbounds int    `json:"inbounds"`
}

type Inbound struct {
	ID        int64  `json:"id"`
	Tag       string `json:"tag"`
	Protocol  string `json:"protocol"`
	Port      uint16 `json:"port"`
	Settings  string `json:"settings"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type TrafficSummary struct {
	Tag      string `json:"tag"`
	Upload   int64  `json:"upload"`
	Download int64  `json:"download"`
}

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) do(method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.http.Do(req)
}

func decode[T any](resp *http.Response) (T, error) {
	var result T
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("failed to decode response: %w", err)
	}
	return result, nil
}

func (c *Client) Status() (*StatusResp, error) {
	resp, err := c.do("GET", "/api/status", nil)
	if err != nil {
		return nil, err
	}
	result, err := decode[StatusResp](resp)
	return &result, err
}

func (c *Client) ListInbounds() ([]Inbound, error) {
	resp, err := c.do("GET", "/api/inbounds", nil)
	if err != nil {
		return nil, err
	}
	return decode[[]Inbound](resp)
}

func (c *Client) CreateInbound(protocol string, port uint16) (*Inbound, error) {
	resp, err := c.do("POST", "/api/inbounds", map[string]any{
		"protocol": protocol,
		"port":     port,
	})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("create failed (%d): %s", resp.StatusCode, body)
	}
	result, err := decode[Inbound](resp)
	return &result, err
}

func (c *Client) GetInbound(id int64) (*Inbound, error) {
	resp, err := c.do("GET", fmt.Sprintf("/api/inbounds/%d", id), nil)
	if err != nil {
		return nil, err
	}
	result, err := decode[Inbound](resp)
	return &result, err
}

func (c *Client) UpdateInbound(id int64, port *uint16, settings *json.RawMessage) (*Inbound, error) {
	body := map[string]any{}
	if port != nil {
		body["port"] = *port
	}
	if settings != nil {
		body["settings"] = settings
	}
	resp, err := c.do("PUT", fmt.Sprintf("/api/inbounds/%d", id), body)
	if err != nil {
		return nil, err
	}
	result, err := decode[Inbound](resp)
	return &result, err
}

func (c *Client) DeleteInbound(id int64) error {
	resp, err := c.do("DELETE", fmt.Sprintf("/api/inbounds/%d", id), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete failed: %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) GetStats() ([]TrafficSummary, error) {
	resp, err := c.do("GET", "/api/stats", nil)
	if err != nil {
		return nil, err
	}
	return decode[[]TrafficSummary](resp)
}

func (c *Client) Reload() error {
	resp, err := c.do("POST", "/api/reload", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
```

- [ ] **Step 3: Run tests**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/tui/ -v -count=1`

Expected: All 5 tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/client.go internal/tui/client_test.go
git commit -m "feat(tui): HTTP API client for TUI"
```

---

### Task 3: QR Terminal Renderer

**Files:**
- Create: `internal/tui/qr.go`
- Create: `internal/tui/qr_test.go`

- [ ] **Step 1: Write QR tests**

Create `internal/tui/qr_test.go`:

```go
package tui

import (
	"strings"
	"testing"
)

func TestRenderQR(t *testing.T) {
	result := RenderQR("https://example.com")
	if result == "" {
		t.Fatal("QR should not be empty")
	}
	lines := strings.Split(result, "\n")
	if len(lines) < 5 {
		t.Errorf("QR too small: %d lines", len(lines))
	}
}

func TestRenderQREmpty(t *testing.T) {
	result := RenderQR("")
	if result != "" {
		t.Error("empty input should return empty string")
	}
}
```

- [ ] **Step 2: Implement qr.go**

Create `internal/tui/qr.go`:

```go
package tui

import (
	"strings"

	"github.com/skip2/go-qrcode"
)

// RenderQR generates a QR code as a string using Unicode half-block characters.
func RenderQR(content string) string {
	if content == "" {
		return ""
	}

	q, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return ""
	}

	bitmap := q.Bitmap()
	var sb strings.Builder

	for y := 0; y < len(bitmap); y += 2 {
		for x := 0; x < len(bitmap[y]); x++ {
			top := bitmap[y][x]
			bot := y+1 < len(bitmap) && bitmap[y+1][x]
			switch {
			case top && bot:
				sb.WriteRune('█')
			case top:
				sb.WriteRune('▀')
			case bot:
				sb.WriteRune('▄')
			default:
				sb.WriteRune(' ')
			}
		}
		sb.WriteRune('\n')
	}

	return sb.String()
}
```

- [ ] **Step 3: Run tests**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/tui/ -v -count=1 -run TestRenderQR`

Expected: Both tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/qr.go internal/tui/qr_test.go
git commit -m "feat(tui): QR code terminal renderer using Unicode half-blocks"
```

---

### Task 4: Lipgloss Styles + TUI Core Model

**Files:**
- Create: `internal/tui/style.go`
- Create: `internal/tui/tui.go`

- [ ] **Step 1: Create style.go**

Create `internal/tui/style.go`:

```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			MarginBottom(1)

	menuStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2).
			Width(44)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))
)
```

- [ ] **Step 2: Create tui.go (core model + state machine)**

Create `internal/tui/tui.go`:

```go
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type viewState int

const (
	stateMenu viewState = iota
	stateAdd
	stateList
	stateDetail
	stateEdit
	stateStats
	stateStatus
)

type app struct {
	state      viewState
	client     *Client
	listenAddr string
	subToken   string

	// View models
	menu   menuModel
	add    addModel
	list   listModel
	detail detailModel
	edit   editModel
	stats  statsModel
	status statusModel

	// Shared state
	err     error
	message string
	width   int
	height  int
}

func newApp(client *Client, listenAddr, subToken string) app {
	return app{
		state:      stateMenu,
		client:     client,
		listenAddr: listenAddr,
		subToken:   subToken,
		menu:       newMenuModel(),
	}
}

func (a app) Init() tea.Cmd {
	return nil
}

func (a app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
	case errMsg:
		a.err = msg.err
		return a, nil
	case clearMsg:
		a.message = ""
		a.err = nil
		return a, nil
	}

	switch a.state {
	case stateMenu:
		return a.updateMenu(msg)
	case stateAdd:
		return a.updateAdd(msg)
	case stateList:
		return a.updateList(msg)
	case stateDetail:
		return a.updateDetail(msg)
	case stateEdit:
		return a.updateEdit(msg)
	case stateStats:
		return a.updateStats(msg)
	case stateStatus:
		return a.updateStatus(msg)
	}

	return a, nil
}

func (a app) View() string {
	var content string

	switch a.state {
	case stateMenu:
		content = a.viewMenu()
	case stateAdd:
		content = a.viewAdd()
	case stateList:
		content = a.viewList()
	case stateDetail:
		content = a.viewDetail()
	case stateEdit:
		content = a.viewEdit()
	case stateStats:
		content = a.viewStats()
	case stateStatus:
		content = a.viewStatus()
	}

	if a.err != nil {
		content += "\n" + errorStyle.Render("错误: "+a.err.Error())
	}
	if a.message != "" {
		content += "\n" + successStyle.Render(a.message)
	}

	return content
}

// Run starts the TUI application.
func Run(listenAddr, token, subToken string) error {
	client := NewClient("http://"+listenAddr, token)
	a := newApp(client, listenAddr, subToken)
	p := tea.NewProgram(a, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Message types
type errMsg struct{ err error }
type clearMsg struct{}

type inboundsLoadedMsg struct{ inbounds []Inbound }
type inboundCreatedMsg struct{ inbound *Inbound }
type inboundDeletedMsg struct{}
type inboundUpdatedMsg struct{ inbound *Inbound }
type statusLoadedMsg struct{ status *StatusResp }
type statsLoadedMsg struct{ stats []TrafficSummary }

func loadInbounds(c *Client) tea.Cmd {
	return func() tea.Msg {
		list, err := c.ListInbounds()
		if err != nil {
			return errMsg{err}
		}
		return inboundsLoadedMsg{list}
	}
}

func loadStatus(c *Client) tea.Cmd {
	return func() tea.Msg {
		s, err := c.Status()
		if err != nil {
			return errMsg{err}
		}
		return statusLoadedMsg{s}
	}
}

func loadStats(c *Client) tea.Cmd {
	return func() tea.Msg {
		stats, err := c.GetStats()
		if err != nil {
			return errMsg{err}
		}
		return statsLoadedMsg{stats}
	}
}

// FormatBytes formats bytes to human-readable string.
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go build ./internal/tui/`

Expected: Compilation error — menuModel and other view types not defined yet. This is expected; they will be implemented in subsequent tasks.

- [ ] **Step 4: Commit style.go (tui.go will be committed with menu.go)**

```bash
git add internal/tui/style.go
git commit -m "feat(tui): Lipgloss style constants"
```

---

### Task 5: Main Menu View

**Files:**
- Create: `internal/tui/menu.go`

- [ ] **Step 1: Implement menu.go**

Create `internal/tui/menu.go`:

```go
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

var menuItems = []string{
	"添加配置",
	"查看配置",
	"修改配置",
	"流量统计",
	"服务状态",
	"退出",
}

type menuModel struct {
	cursor int
}

func newMenuModel() menuModel {
	return menuModel{}
}

func (a app) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return a, tea.Quit
		case "up", "k":
			if a.menu.cursor > 0 {
				a.menu.cursor--
			}
		case "down", "j":
			if a.menu.cursor < len(menuItems)-1 {
				a.menu.cursor++
			}
		case "enter":
			return a.menuAction()
		}
	}
	return a, nil
}

func (a app) menuAction() (tea.Model, tea.Cmd) {
	switch a.menu.cursor {
	case 0: // 添加配置
		a.state = stateAdd
		a.add = newAddModel()
		return a, a.add.init()
	case 1: // 查看配置
		a.state = stateList
		a.list = newListModel()
		return a, loadInbounds(a.client)
	case 2: // 修改配置
		a.state = stateList
		a.list = newListModel()
		a.list.editMode = true
		return a, loadInbounds(a.client)
	case 3: // 流量统计
		a.state = stateStats
		a.stats = newStatsModel()
		return a, loadStats(a.client)
	case 4: // 服务状态
		a.state = stateStatus
		a.status = newStatusModel()
		return a, loadStatus(a.client)
	case 5: // 退出
		return a, tea.Quit
	}
	return a, nil
}

func (a app) viewMenu() string {
	title := titleStyle.Render("sing-box 管理面板")

	var items string
	for i, item := range menuItems {
		cursor := "  "
		style := normalStyle
		if a.menu.cursor == i {
			cursor = "> "
			style = selectedStyle
		}
		items += fmt.Sprintf("%s%s\n", cursor, style.Render(item))
	}

	return menuStyle.Render(title + "\n" + items + "\n" + dimStyle.Render("↑↓ 选择  Enter 确认  q 退出"))
}
```

- [ ] **Step 2: Create stub files for all remaining views**

Since `tui.go` references all view types, we need stubs to compile. Create temporary stubs that will be replaced in later tasks.

Create `internal/tui/add.go`:

```go
package tui

import tea "github.com/charmbracelet/bubbletea"

type addModel struct {
	step     int
	protocol int
	port     string
	result   *Inbound
	err      error
}

func newAddModel() addModel { return addModel{} }
func (m addModel) init() tea.Cmd { return nil }
func (a app) updateAdd(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "esc" {
		a.state = stateMenu
	}
	return a, nil
}
func (a app) viewAdd() string { return "添加配置（开发中）\n\nEsc 返回" }
```

Create `internal/tui/list.go`:

```go
package tui

import tea "github.com/charmbracelet/bubbletea"

type listModel struct {
	inbounds []Inbound
	cursor   int
	editMode bool
}

func newListModel() listModel { return listModel{} }

type detailModel struct {
	inbound  *Inbound
	shareURL string
	confirm  bool
}

func (a app) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case inboundsLoadedMsg:
		a.list.inbounds = msg.inbounds
	case tea.KeyMsg:
		if msg.String() == "esc" {
			a.state = stateMenu
		}
	}
	return a, nil
}
func (a app) viewList() string { return "配置列表（开发中）\n\nEsc 返回" }
func (a app) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "esc" {
		a.state = stateList
	}
	return a, nil
}
func (a app) viewDetail() string { return "配置详情（开发中）\n\nEsc 返回" }
```

Create `internal/tui/edit.go`:

```go
package tui

import tea "github.com/charmbracelet/bubbletea"

type editModel struct {
	inbound *Inbound
	port    string
	err     error
}

func (a app) updateEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "esc" {
		a.state = stateList
	}
	return a, nil
}
func (a app) viewEdit() string { return "修改配置（开发中）\n\nEsc 返回" }
```

Create `internal/tui/stats.go`:

```go
package tui

import tea "github.com/charmbracelet/bubbletea"

type statsModel struct {
	stats []TrafficSummary
}

func newStatsModel() statsModel { return statsModel{} }
func (a app) updateStats(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case statsLoadedMsg:
		a.stats.stats = msg.stats
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			a.state = stateMenu
		case "r":
			return a, loadStats(a.client)
		}
	}
	return a, nil
}
func (a app) viewStats() string { return "流量统计（开发中）\n\nEsc 返回" }
```

Create `internal/tui/status.go`:

```go
package tui

import tea "github.com/charmbracelet/bubbletea"

type statusModel struct {
	status *StatusResp
}

func newStatusModel() statusModel { return statusModel{} }
func (a app) updateStatus(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case statusLoadedMsg:
		a.status.status = msg.status
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			a.state = stateMenu
		case "r":
			return a, loadStatus(a.client)
		}
	}
	return a, nil
}
func (a app) viewStatus() string { return "服务状态（开发中）\n\nEsc 返回" }
```

- [ ] **Step 3: Verify compilation**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go build ./internal/tui/`

Expected: Compiles successfully.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/tui.go internal/tui/menu.go internal/tui/add.go internal/tui/list.go internal/tui/edit.go internal/tui/stats.go internal/tui/status.go
git commit -m "feat(tui): core app model, state machine, and main menu"
```

---

### Task 6: Add Inbound Wizard

**Files:**
- Modify: `internal/tui/add.go` (replace stub)

- [ ] **Step 1: Replace add.go with full implementation**

Replace `internal/tui/add.go`:

```go
package tui

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
)

var protocols = []string{"shadowsocks", "vless", "trojan"}
var protocolNames = []string{"Shadowsocks (SS2022)", "VLESS-REALITY", "Trojan"}

type addModel struct {
	step     int // 0=protocol, 1=port, 2=confirm, 3=result
	protocol int
	portInput textinput.Model
	result   *Inbound
	err      error
}

func newAddModel() addModel {
	ti := textinput.New()
	ti.Placeholder = "端口号 (1-65535)"
	ti.CharLimit = 5
	ti.Width = 20
	return addModel{portInput: ti}
}

func (m addModel) init() tea.Cmd {
	return nil
}

func (a app) updateAdd(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case inboundCreatedMsg:
		a.add.result = msg.inbound
		a.add.step = 3
		return a, nil
	case errMsg:
		a.add.err = msg.err
		return a, nil
	case tea.KeyMsg:
		switch a.add.step {
		case 0: // Select protocol
			switch msg.String() {
			case "esc":
				a.state = stateMenu
				return a, nil
			case "up", "k":
				if a.add.protocol > 0 {
					a.add.protocol--
				}
			case "down", "j":
				if a.add.protocol < len(protocols)-1 {
					a.add.protocol++
				}
			case "enter":
				a.add.step = 1
				a.add.portInput.Focus()
				return a, textinput.Blink
			}
			return a, nil
		case 1: // Enter port
			switch msg.String() {
			case "esc":
				a.add.step = 0
				return a, nil
			case "enter":
				port, err := strconv.Atoi(a.add.portInput.Value())
				if err != nil || port < 1 || port > 65535 {
					a.add.err = fmt.Errorf("端口号必须在 1-65535 之间")
					return a, nil
				}
				a.add.err = nil
				a.add.step = 2
				return a, nil
			}
			var cmd tea.Cmd
			a.add.portInput, cmd = a.add.portInput.Update(msg)
			return a, cmd
		case 2: // Confirm
			switch msg.String() {
			case "esc":
				a.add.step = 1
				return a, nil
			case "enter":
				port, _ := strconv.Atoi(a.add.portInput.Value())
				proto := protocols[a.add.protocol]
				return a, createInbound(a.client, proto, uint16(port))
			}
			return a, nil
		case 3: // Result
			a.state = stateMenu
			a.message = "配置添加成功"
			return a, nil
		}
	}
	return a, nil
}

func createInbound(c *Client, protocol string, port uint16) tea.Cmd {
	return func() tea.Msg {
		ib, err := c.CreateInbound(protocol, port)
		if err != nil {
			return errMsg{err}
		}
		return inboundCreatedMsg{ib}
	}
}

func (a app) viewAdd() string {
	title := titleStyle.Render("添加配置")

	switch a.add.step {
	case 0:
		var items string
		for i, name := range protocolNames {
			cursor := "  "
			style := normalStyle
			if a.add.protocol == i {
				cursor = "> "
				style = selectedStyle
			}
			items += fmt.Sprintf("%s%s\n", cursor, style.Render(name))
		}
		content := title + "\n\n选择协议:\n\n" + items
		if a.add.err != nil {
			content += "\n" + errorStyle.Render(a.add.err.Error())
		}
		return menuStyle.Render(content + "\n" + dimStyle.Render("↑↓ 选择  Enter 确认  Esc 返回"))

	case 1:
		content := title + "\n\n协议: " + infoStyle.Render(protocolNames[a.add.protocol]) + "\n\n端口: " + a.add.portInput.View()
		if a.add.err != nil {
			content += "\n\n" + errorStyle.Render(a.add.err.Error())
		}
		return menuStyle.Render(content + "\n\n" + dimStyle.Render("Enter 确认  Esc 返回"))

	case 2:
		content := title + "\n\n" +
			"协议: " + infoStyle.Render(protocolNames[a.add.protocol]) + "\n" +
			"端口: " + infoStyle.Render(a.add.portInput.Value()) + "\n\n" +
			selectedStyle.Render("按 Enter 确认创建")
		return menuStyle.Render(content + "\n\n" + dimStyle.Render("Enter 创建  Esc 返回"))

	case 3:
		ib := a.add.result
		shareURL := a.getShareURL(ib)
		content := title + "\n\n" +
			successStyle.Render("✓ 创建成功") + "\n\n" +
			"Tag:  " + infoStyle.Render(ib.Tag) + "\n" +
			"端口: " + infoStyle.Render(fmt.Sprintf("%d", ib.Port)) + "\n\n" +
			"分享链接:\n" + dimStyle.Render(shareURL) + "\n\n" +
			RenderQR(shareURL)
		return menuStyle.Width(60).Render(content + dimStyle.Render("按任意键返回"))
	}
	return ""
}

func (a app) getShareURL(ib *Inbound) string {
	// Use protocol package to generate URL
	// Since TUI is an API client, we generate URL client-side using settings
	host := a.listenAddr
	// Extract IP without port for share URLs
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			host = host[:i]
			break
		}
	}
	if host == "127.0.0.1" || host == "0.0.0.0" {
		host = "YOUR_SERVER_IP"
	}

	// Import protocol package for URL generation
	return generateShareURL(ib, host)
}
```

We also need a helper in tui.go to generate share URLs. Add to `internal/tui/tui.go` before the closing of the file:

```go
import (
	"github.com/233boy/sing-box/internal/protocol"
	"github.com/233boy/sing-box/internal/store"
)

func generateShareURL(ib *Inbound, host string) string {
	p := protocol.Get(ib.Protocol)
	if p == nil {
		return ""
	}
	storeIb := &store.Inbound{
		ID:       ib.ID,
		Tag:      ib.Tag,
		Protocol: ib.Protocol,
		Port:     ib.Port,
		Settings: ib.Settings,
	}
	return p.GenerateURL(storeIb, host)
}
```

Note: This requires adding the protocol and store imports to tui.go.

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go build ./internal/tui/`

Expected: Compiles successfully.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/add.go internal/tui/tui.go
git commit -m "feat(tui): add inbound wizard with protocol selection and QR display"
```

---

### Task 7: Inbound List + Detail View

**Files:**
- Modify: `internal/tui/list.go` (replace stub)

- [ ] **Step 1: Replace list.go with full implementation**

Replace `internal/tui/list.go`:

```go
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type listModel struct {
	inbounds []Inbound
	cursor   int
	editMode bool
}

func newListModel() listModel {
	return listModel{}
}

type detailModel struct {
	inbound  *Inbound
	shareURL string
	confirm  bool
}

func (a app) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case inboundsLoadedMsg:
		a.list.inbounds = msg.inbounds
		a.list.cursor = 0
	case inboundDeletedMsg:
		a.message = "配置已删除"
		return a, loadInbounds(a.client)
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			a.state = stateMenu
			a.list.editMode = false
			return a, nil
		case "up", "k":
			if a.list.cursor > 0 {
				a.list.cursor--
			}
		case "down", "j":
			if a.list.cursor < len(a.list.inbounds)-1 {
				a.list.cursor++
			}
		case "enter":
			if len(a.list.inbounds) > 0 {
				ib := a.list.inbounds[a.list.cursor]
				if a.list.editMode {
					a.state = stateEdit
					a.edit = editModel{inbound: &ib, port: fmt.Sprintf("%d", ib.Port)}
					return a, nil
				}
				host := extractHost(a.listenAddr)
				a.detail = detailModel{
					inbound:  &ib,
					shareURL: generateShareURL(&ib, host),
				}
				a.state = stateDetail
			}
			return a, nil
		case "r":
			return a, loadInbounds(a.client)
		}
	}
	return a, nil
}

func (a app) viewList() string {
	modeLabel := "查看配置"
	if a.list.editMode {
		modeLabel = "修改配置"
	}
	title := titleStyle.Render(modeLabel)

	if len(a.list.inbounds) == 0 {
		return menuStyle.Render(title + "\n\n" + dimStyle.Render("暂无配置") + "\n\n" + dimStyle.Render("Esc 返回"))
	}

	var items string
	for i, ib := range a.list.inbounds {
		cursor := "  "
		style := normalStyle
		if a.list.cursor == i {
			cursor = "> "
			style = selectedStyle
		}
		items += fmt.Sprintf("%s%s\n", cursor, style.Render(
			fmt.Sprintf("%-15s  端口:%-6d  %s", ib.Protocol, ib.Port, ib.Tag),
		))
	}

	hint := "↑↓ 选择  Enter 详情  r 刷新  Esc 返回"
	if a.list.editMode {
		hint = "↑↓ 选择  Enter 编辑  r 刷新  Esc 返回"
	}
	return menuStyle.Width(60).Render(title + "\n\n" + items + "\n" + dimStyle.Render(hint))
}

func (a app) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case inboundDeletedMsg:
		a.state = stateList
		a.message = "配置已删除"
		return a, loadInbounds(a.client)
	case tea.KeyMsg:
		if a.detail.confirm {
			switch msg.String() {
			case "y":
				a.detail.confirm = false
				return a, deleteInbound(a.client, a.detail.inbound.ID)
			default:
				a.detail.confirm = false
				return a, nil
			}
		}
		switch msg.String() {
		case "esc":
			a.state = stateList
			return a, nil
		case "d":
			a.detail.confirm = true
			return a, nil
		case "e":
			ib := a.detail.inbound
			a.state = stateEdit
			a.edit = editModel{inbound: ib, port: fmt.Sprintf("%d", ib.Port)}
			return a, nil
		}
	}
	return a, nil
}

func deleteInbound(c *Client, id int64) tea.Cmd {
	return func() tea.Msg {
		if err := c.DeleteInbound(id); err != nil {
			return errMsg{err}
		}
		return inboundDeletedMsg{}
	}
}

func (a app) viewDetail() string {
	ib := a.detail.inbound
	title := titleStyle.Render("配置详情")

	info := fmt.Sprintf(
		"Tag:      %s\n协议:     %s\n端口:     %d\n",
		infoStyle.Render(ib.Tag),
		infoStyle.Render(ib.Protocol),
		ib.Port,
	)

	content := title + "\n\n" + info + "\n"

	if a.detail.shareURL != "" {
		content += "分享链接:\n" + dimStyle.Render(a.detail.shareURL) + "\n\n"
		content += RenderQR(a.detail.shareURL)
	}

	if a.detail.confirm {
		content += "\n" + errorStyle.Render("确认删除? (y/n)")
	}

	return menuStyle.Width(60).Render(content + "\n" + dimStyle.Render("d 删除  e 编辑  Esc 返回"))
}

func extractHost(listenAddr string) string {
	host := listenAddr
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			host = host[:i]
			break
		}
	}
	if host == "127.0.0.1" || host == "0.0.0.0" || host == "" {
		host = "YOUR_SERVER_IP"
	}
	return host
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go build ./internal/tui/`

Expected: Compiles successfully.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/list.go
git commit -m "feat(tui): inbound list and detail view with QR code display"
```

---

### Task 8: Edit View

**Files:**
- Modify: `internal/tui/edit.go` (replace stub)

- [ ] **Step 1: Replace edit.go**

Replace `internal/tui/edit.go`:

```go
package tui

import (
	"encoding/json"
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
)

type editModel struct {
	inbound   *Inbound
	port      string
	portInput textinput.Model
	focused   bool
	err       error
}

func newEditModel(ib *Inbound) editModel {
	ti := textinput.New()
	ti.SetValue(fmt.Sprintf("%d", ib.Port))
	ti.CharLimit = 5
	ti.Width = 20
	ti.Focus()
	return editModel{
		inbound:   ib,
		port:      fmt.Sprintf("%d", ib.Port),
		portInput: ti,
		focused:   true,
	}
}

func (a app) updateEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case inboundUpdatedMsg:
		a.state = stateList
		a.message = "配置已更新"
		return a, loadInbounds(a.client)
	case errMsg:
		a.edit.err = msg.err
		return a, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			a.state = stateList
			return a, nil
		case "enter":
			port, err := strconv.Atoi(a.edit.portInput.Value())
			if err != nil || port < 1 || port > 65535 {
				a.edit.err = fmt.Errorf("端口号必须在 1-65535 之间")
				return a, nil
			}
			p := uint16(port)
			return a, updateInbound(a.client, a.edit.inbound.ID, &p)
		}
		var cmd tea.Cmd
		a.edit.portInput, cmd = a.edit.portInput.Update(msg)
		return a, cmd
	}
	return a, nil
}

func updateInbound(c *Client, id int64, port *uint16) tea.Cmd {
	return func() tea.Msg {
		var settings *json.RawMessage
		ib, err := c.UpdateInbound(id, port, settings)
		if err != nil {
			return errMsg{err}
		}
		return inboundUpdatedMsg{ib}
	}
}

func (a app) viewEdit() string {
	ib := a.edit.inbound
	title := titleStyle.Render("修改配置")

	content := title + "\n\n" +
		"Tag:  " + infoStyle.Render(ib.Tag) + "\n" +
		"协议: " + infoStyle.Render(ib.Protocol) + "\n\n" +
		"端口: " + a.edit.portInput.View()

	if a.edit.err != nil {
		content += "\n\n" + errorStyle.Render(a.edit.err.Error())
	}

	return menuStyle.Render(content + "\n\n" + dimStyle.Render("Enter 保存  Esc 取消"))
}
```

- [ ] **Step 2: Verify + Commit**

Run: `go build ./internal/tui/`

```bash
git add internal/tui/edit.go
git commit -m "feat(tui): edit inbound view with port modification"
```

---

### Task 9: Stats + Status Views

**Files:**
- Modify: `internal/tui/stats.go` (replace stub)
- Modify: `internal/tui/status.go` (replace stub)

- [ ] **Step 1: Replace stats.go**

Replace `internal/tui/stats.go`:

```go
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type statsModel struct {
	stats []TrafficSummary
}

func newStatsModel() statsModel { return statsModel{} }

func (a app) updateStats(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case statsLoadedMsg:
		a.stats.stats = msg.stats
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			a.state = stateMenu
			return a, nil
		case "r":
			return a, loadStats(a.client)
		}
	}
	return a, nil
}

func (a app) viewStats() string {
	title := titleStyle.Render("流量统计")

	if len(a.stats.stats) == 0 {
		return menuStyle.Render(title + "\n\n" + dimStyle.Render("暂无流量数据") + "\n\n" + dimStyle.Render("r 刷新  Esc 返回"))
	}

	header := fmt.Sprintf("  %-20s  %12s  %12s\n", "Tag", "上传", "下载")
	header += fmt.Sprintf("  %-20s  %12s  %12s\n", "───────────────────", "──────────", "──────────")

	var rows string
	for _, s := range a.stats.stats {
		rows += fmt.Sprintf("  %-20s  %12s  %12s\n",
			s.Tag, FormatBytes(s.Upload), FormatBytes(s.Download))
	}

	return menuStyle.Width(60).Render(title + "\n\n" + infoStyle.Render(header) + rows + "\n" + dimStyle.Render("r 刷新  Esc 返回"))
}
```

- [ ] **Step 2: Replace status.go**

Replace `internal/tui/status.go`:

```go
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type statusModel struct {
	status *StatusResp
}

func newStatusModel() statusModel { return statusModel{} }

func (a app) updateStatus(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case statusLoadedMsg:
		a.status.status = msg.status
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			a.state = stateMenu
			return a, nil
		case "r":
			return a, loadStatus(a.client)
		}
	}
	return a, nil
}

func (a app) viewStatus() string {
	title := titleStyle.Render("服务状态")

	if a.status.status == nil {
		return menuStyle.Render(title + "\n\n" + dimStyle.Render("加载中..."))
	}

	s := a.status.status
	engineStatus := errorStyle.Render("已停止")
	if s.Engine == "running" {
		engineStatus = successStyle.Render("运行中")
	}

	subURL := fmt.Sprintf("http://%s/sub/%s", a.listenAddr, a.subToken)

	content := title + "\n\n" +
		"引擎状态: " + engineStatus + "\n" +
		"配置数量: " + infoStyle.Render(fmt.Sprintf("%d", s.Inbounds)) + "\n" +
		"API 地址: " + dimStyle.Render("http://"+a.listenAddr) + "\n\n" +
		"订阅地址:\n" + infoStyle.Render(subURL) + "\n\n" +
		RenderQR(subURL)

	return menuStyle.Width(60).Render(content + dimStyle.Render("r 刷新  Esc 返回"))
}
```

- [ ] **Step 3: Verify + Commit**

Run: `go build ./internal/tui/`

```bash
git add internal/tui/stats.go internal/tui/status.go
git commit -m "feat(tui): traffic stats table and engine status view"
```

---

### Task 10: Updater Package

**Files:**
- Create: `internal/updater/updater.go`
- Create: `internal/updater/updater_test.go`

- [ ] **Step 1: Write updater tests**

Create `internal/updater/updater_test.go`:

```go
package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckLatestVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"tag_name": "v0.2.0"})
	}))
	defer ts.Close()

	v, url, err := checkLatestVersion(ts.URL + "/")
	if err != nil {
		t.Fatalf("checkLatestVersion error: %v", err)
	}
	if v != "v0.2.0" {
		t.Errorf("version = %q, want v0.2.0", v)
	}
	if url == "" {
		t.Error("download URL should not be empty")
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		current, latest string
		want            bool
	}{
		{"v0.1.0", "v0.2.0", true},
		{"v0.2.0", "v0.2.0", false},
		{"v0.3.0", "v0.2.0", false},
		{"dev", "v0.1.0", true},
	}
	for _, tt := range tests {
		got := isNewer(tt.current, tt.latest)
		if got != tt.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
		}
	}
}
```

- [ ] **Step 2: Implement updater.go**

Create `internal/updater/updater.go`:

```go
package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

const defaultAPIBase = "https://api.github.com/repos/233boy/sing-box/releases/latest"

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// Update checks for the latest version and replaces the current binary.
func Update(currentVersion string) error {
	fmt.Println("检查最新版本...")

	latest, downloadURL, err := checkLatestVersion(defaultAPIBase)
	if err != nil {
		return fmt.Errorf("检查更新失败: %w", err)
	}

	if !isNewer(currentVersion, latest) {
		fmt.Printf("当前已是最新版本 (%s)\n", currentVersion)
		return nil
	}

	fmt.Printf("发现新版本: %s → %s\n", currentVersion, latest)
	fmt.Println("下载中...")

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("无法获取当前二进制路径: %w", err)
	}

	tmpPath := execPath + ".tmp"
	if err := downloadFile(downloadURL, tmpPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("下载失败: %w", err)
	}

	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("设置权限失败: %w", err)
	}

	if err := os.Rename(tmpPath, execPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("替换二进制失败: %w", err)
	}

	fmt.Printf("更新完成: %s → %s\n", currentVersion, latest)
	fmt.Println("请运行 sing-box service restart 完成更新")
	return nil
}

func checkLatestVersion(apiURL string) (version string, downloadURL string, err error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	archName := fmt.Sprintf("sing-box-linux-%s", runtime.GOARCH)
	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, archName) {
			return release.TagName, asset.BrowserDownloadURL, nil
		}
	}

	// Fallback: construct URL
	downloadURL = fmt.Sprintf("https://github.com/233boy/sing-box/releases/download/%s/%s", release.TagName, archName)
	return release.TagName, downloadURL, nil
}

func downloadFile(url, dest string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func isNewer(current, latest string) bool {
	if current == "dev" || current == "" {
		return true
	}
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")
	return latest > current
}
```

- [ ] **Step 3: Run tests**

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/updater/ -v -count=1`

Expected: All tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/updater/updater.go internal/updater/updater_test.go
git commit -m "feat(updater): self-update via GitHub Releases with atomic binary replace"
```

---

### Task 11: Main Entry Point — TUI Mode + New Commands

**Files:**
- Modify: `cmd/sing-box/main.go`

- [ ] **Step 1: Rewrite main.go**

Replace `cmd/sing-box/main.go`:

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
	"runtime"
	"syscall"
	"time"

	"github.com/233boy/sing-box/internal/api"
	"github.com/233boy/sing-box/internal/engine"
	"github.com/233boy/sing-box/internal/platform"
	"github.com/233boy/sing-box/internal/service"
	"github.com/233boy/sing-box/internal/stats"
	"github.com/233boy/sing-box/internal/store"
	"github.com/233boy/sing-box/internal/tui"
	"github.com/233boy/sing-box/internal/updater"
	"github.com/spf13/cobra"

	_ "github.com/233boy/sing-box/internal/protocol"
)

var Version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "sing-box",
		Short: "sing-box 管理面板",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, _ := cmd.Flags().GetString("listen")
			dataDir, _ := cmd.Flags().GetString("data-dir")
			return runTUI(listenAddr, dataDir)
		},
	}

	plat := platform.Detect()
	rootCmd.Flags().String("listen", "127.0.0.1:9090", "API 监听地址")
	rootCmd.Flags().String("data-dir", plat.DataDir, "数据目录")

	rootCmd.AddCommand(serveCmd())
	rootCmd.AddCommand(updateCmd())
	rootCmd.AddCommand(serviceCmd())
	rootCmd.AddCommand(versionCmd())

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

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "检查并更新到最新版本",
		RunE: func(cmd *cobra.Command, args []string) error {
			return updater.Update(Version)
		},
	}
}

func serviceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "管理系统服务",
	}

	plat := platform.Detect()

	cmd.AddCommand(&cobra.Command{
		Use:   "install",
		Short: "安装系统服务",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := service.NewManager(plat.InitSystem)
			if mgr == nil {
				return fmt.Errorf("不支持的 init 系统: %s", plat.InitSystem)
			}
			if err := mgr.Install(plat.BinPath, plat.DataDir); err != nil {
				return fmt.Errorf("安装服务失败: %w", err)
			}
			fmt.Println("服务已安装并启用")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "uninstall",
		Short: "卸载系统服务",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := service.NewManager(plat.InitSystem)
			if mgr == nil {
				return fmt.Errorf("不支持的 init 系统: %s", plat.InitSystem)
			}
			if err := mgr.Uninstall(); err != nil {
				return fmt.Errorf("卸载服务失败: %w", err)
			}
			fmt.Println("服务已卸载")
			return nil
		},
	})

	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("sing-box panel %s (%s %s/%s)\n", Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		},
	}
}

func runTUI(listenAddr, dataDir string) error {
	// Start server components (same as runServe but with TUI)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("无法创建数据目录: %w", err)
	}

	dbPath := dataDir + "/panel.db"
	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("无法打开数据库: %w", err)
	}
	defer st.Close()

	apiToken, err := ensureToken(st, "api_token")
	if err != nil {
		return fmt.Errorf("无法初始化 API token: %w", err)
	}

	subToken, err := ensureToken(st, "sub_token")
	if err != nil {
		return fmt.Errorf("无法初始化订阅 token: %w", err)
	}

	plat := platform.Detect()
	var svcMgr service.Manager
	if mgr := service.NewManager(plat.InitSystem); mgr != nil {
		svcMgr = mgr
	}

	eng := engine.New(st)
	if err := eng.Start(); err != nil {
		slog.Warn("引擎启动失败（可能没有配置）", "error", err)
	}

	collector := stats.NewCollector(eng.Tracker(), st)
	collectorCtx, collectorCancel := context.WithCancel(context.Background())
	go collector.Run(collectorCtx, 60*time.Second)

	srv := api.NewServer(eng, st, svcMgr, listenAddr, apiToken, subToken)
	if err := srv.Start(); err != nil {
		collectorCancel()
		return fmt.Errorf("无法启动 API 服务: %w", err)
	}

	// Run TUI (blocks until exit)
	tuiErr := tui.Run(listenAddr, apiToken, subToken)

	// Graceful shutdown
	collectorCancel()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
	eng.Stop()

	return tuiErr
}

func runServe(listenAddr, dataDir string) error {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("无法创建数据目录: %w", err)
	}

	dbPath := dataDir + "/panel.db"
	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("无法打开数据库: %w", err)
	}
	defer st.Close()

	apiToken, err := ensureToken(st, "api_token")
	if err != nil {
		return fmt.Errorf("无法初始化 API token: %w", err)
	}

	subToken, err := ensureToken(st, "sub_token")
	if err != nil {
		return fmt.Errorf("无法初始化订阅 token: %w", err)
	}

	plat := platform.Detect()
	var svcMgr service.Manager
	if mgr := service.NewManager(plat.InitSystem); mgr != nil {
		svcMgr = mgr
	}

	eng := engine.New(st)
	if err := eng.Start(); err != nil {
		slog.Warn("引擎启动失败（可能没有配置）", "error", err)
	}

	collector := stats.NewCollector(eng.Tracker(), st)
	collectorCtx, collectorCancel := context.WithCancel(context.Background())
	go collector.Run(collectorCtx, 60*time.Second)

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

Run: `cd /Users/admin/Codes/ProxyCode/sing-box && go build -o sing-box-m3 ./cmd/sing-box/`

Expected: Compiles successfully.

- [ ] **Step 3: Verify commands**

```bash
./sing-box-m3 --help
./sing-box-m3 version
./sing-box-m3 serve --help
./sing-box-m3 service --help
./sing-box-m3 update --help
```

Expected: All help outputs display correctly. `version` shows `sing-box panel dev`.

- [ ] **Step 4: Commit**

```bash
git add cmd/sing-box/main.go
git commit -m "feat: M3 — TUI mode, update/service/version commands"
```

---

### Task 12: install.sh + .goreleaser.yml

**Files:**
- Create: `install.sh`
- Create: `.goreleaser.yml`

- [ ] **Step 1: Create install.sh**

Create `install.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

# sing-box panel 安装脚本
# 用法: bash <(curl -sL https://raw.githubusercontent.com/233boy/sing-box/main/install.sh)

REPO="233boy/sing-box"
INSTALL_DIR="/usr/local/bin"
DATA_DIR="/usr/local/etc/sing-box"
BIN_NAME="sing-box"
LOCAL_INSTALL=false
VERSION=""
PROXY=""

usage() {
    echo "用法: bash install.sh [选项]"
    echo ""
    echo "选项:"
    echo "  -l          本地安装（从当前目录编译）"
    echo "  -v VERSION  指定版本（如 v0.1.0）"
    echo "  -p PROXY    使用代理下载（如 http://127.0.0.1:2333）"
    echo "  -h          显示帮助"
}

detect_platform() {
    local os arch
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)

    if [[ "$os" != "linux" ]]; then
        echo "错误: 仅支持 Linux 系统" >&2
        exit 1
    fi

    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *)
            echo "错误: 不支持的架构 $arch" >&2
            exit 1
            ;;
    esac

    echo "$os/$arch"
}

get_latest_version() {
    local url="https://api.github.com/repos/$REPO/releases/latest"
    if [[ -n "$PROXY" ]]; then
        curl -sL --proxy "$PROXY" "$url" | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name":\s*"([^"]+)".*/\1/'
    else
        curl -sL "$url" | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name":\s*"([^"]+)".*/\1/'
    fi
}

download_binary() {
    local version="$1" arch="$2"
    local filename="${BIN_NAME}-linux-${arch}"
    local url="https://github.com/$REPO/releases/download/$version/$filename"

    echo "下载 $url ..."

    local curl_opts=(-sL --fail -o "$INSTALL_DIR/$BIN_NAME")
    if [[ -n "$PROXY" ]]; then
        curl_opts+=(--proxy "$PROXY")
    fi

    if ! curl "${curl_opts[@]}" "$url"; then
        echo "错误: 下载失败" >&2
        exit 1
    fi

    chmod +x "$INSTALL_DIR/$BIN_NAME"
}

local_install() {
    echo "本地编译安装..."
    if ! command -v go &>/dev/null; then
        echo "错误: 未找到 go 命令" >&2
        exit 1
    fi
    CGO_ENABLED=0 go build -o "$INSTALL_DIR/$BIN_NAME" ./cmd/sing-box/
    chmod +x "$INSTALL_DIR/$BIN_NAME"
}

main() {
    while getopts "lv:p:h" opt; do
        case "$opt" in
            l) LOCAL_INSTALL=true ;;
            v) VERSION="$OPTARG" ;;
            p) PROXY="$OPTARG" ;;
            h) usage; exit 0 ;;
            *) usage; exit 1 ;;
        esac
    done

    if [[ $EUID -ne 0 ]]; then
        echo "错误: 请使用 root 权限运行" >&2
        exit 1
    fi

    local platform
    platform=$(detect_platform)
    local arch="${platform#*/}"
    echo "平台: $platform"

    if [[ "$LOCAL_INSTALL" == true ]]; then
        local_install
    else
        if [[ -z "$VERSION" ]]; then
            VERSION=$(get_latest_version)
            if [[ -z "$VERSION" ]]; then
                echo "错误: 无法获取最新版本" >&2
                exit 1
            fi
        fi
        echo "版本: $VERSION"
        download_binary "$VERSION" "$arch"
    fi

    # 创建数据目录
    mkdir -p "$DATA_DIR"

    # 安装系统服务
    echo "安装系统服务..."
    "$INSTALL_DIR/$BIN_NAME" service install

    # 启动服务
    echo "启动服务..."
    if command -v systemctl &>/dev/null; then
        systemctl start sing-box
    elif command -v rc-service &>/dev/null; then
        rc-service sing-box start
    fi

    sleep 2

    # 显示信息
    echo ""
    echo "========================================="
    echo " sing-box panel 安装完成"
    echo "========================================="
    echo ""
    echo "管理命令: sing-box"
    echo "守护进程: sing-box serve"
    echo "查看状态: sing-box version"
    echo "更新版本: sing-box update"
    echo ""
}

main "$@"
```

- [ ] **Step 2: Create .goreleaser.yml**

Create `.goreleaser.yml`:

```yaml
version: 2

project_name: sing-box

builds:
  - main: ./cmd/sing-box
    binary: sing-box-{{ .Os }}-{{ .Arch }}
    no_unique_dist_dir: true
    env:
      - CGO_ENABLED=0
    goos:
      - linux
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X main.Version={{.Version}}

archives:
  - format: binary
    name_template: "{{ .Binary }}"

checksum:
  name_template: "checksums.txt"

release:
  github:
    owner: 233boy
    name: sing-box
  draft: false
  prerelease: auto

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^chore:"
```

- [ ] **Step 3: Commit**

```bash
chmod +x install.sh
git add install.sh .goreleaser.yml
git commit -m "feat: install.sh and goreleaser config for v0.1 distribution"
```

---

### Task 13: Full Test Suite + Verification

- [ ] **Step 1: Run full test suite**

```bash
cd /Users/admin/Codes/ProxyCode/sing-box && go test ./internal/... -v -count=1
```

Expected: All tests PASS across all packages.

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

- [ ] **Step 4: Verify version flag**

```bash
go run -ldflags "-X main.Version=v0.1.0" ./cmd/sing-box/ version
```

Expected: `sing-box panel v0.1.0 (go1.26.1 darwin/arm64)`

- [ ] **Step 5: Clean up and commit**

```bash
rm -f sing-box-m3
git add .gitignore
git commit -m "chore: M3 verification complete"
```

---

## Summary

| Package | New/Modified | Key Additions |
|---------|-------------|---------------|
| `tui` | **New** (12 files) | Bubbletea TUI, API client, QR renderer, all views |
| `updater` | **New** (2 files) | GitHub Releases self-update with atomic replace |
| `main` | Modified | TUI mode (root cmd), update/service/version commands |
| root | **New** | install.sh, .goreleaser.yml |

## Cobra Commands

| Command | Description |
|---------|-------------|
| `sing-box` | TUI 模式（内嵌 server） |
| `sing-box serve` | 守护进程模式 |
| `sing-box update` | 自更新 |
| `sing-box service install` | 安装系统服务 |
| `sing-box service uninstall` | 卸载系统服务 |
| `sing-box version` | 显示版本 |

## New Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/charmbracelet/bubbletea` | v1.3.10 | TUI framework |
| `github.com/charmbracelet/lipgloss` | v1.1.0 | Terminal styling |
| `github.com/charmbracelet/bubbles` | v1.0.0 | Text input component |
| `github.com/skip2/go-qrcode` | latest | QR code generation |
