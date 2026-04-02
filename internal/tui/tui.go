package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/cosaria/sing-box/internal/protocol"
	"github.com/cosaria/sing-box/internal/store"
)

type viewState int

const (
	stateMenu   viewState = iota
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
	menu       menuModel
	add        addModel
	list       listModel
	detail     detailModel
	edit       editModel
	stats      statsModel
	status     statusModel
	err        error
	message    string
	width      int
	height     int
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

func (a app) Init() tea.Cmd { return nil }

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

// Run 启动 TUI 应用程序。
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

// FormatBytes 将字节数格式化为人类可读的字符串。
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
