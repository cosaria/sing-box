package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

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
	state   viewState
	store   *store.Store
	dataDir string // 用于查找 PID 文件

	menu   menuModel
	add    addModel
	list   listModel
	detail detailModel
	edit   editModel
	stats  statsModel
	status statusModel

	err     error
	message string
	width   int
	height  int
}

func newApp(st *store.Store, dataDir string) app {
	return app{
		state:   stateMenu,
		store:   st,
		dataDir: dataDir,
		menu:    newMenuModel(),
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
func Run(st *store.Store, dataDir string) error {
	a := newApp(st, dataDir)
	p := tea.NewProgram(a, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Message types
type errMsg struct{ err error }
type clearMsg struct{}
type inboundsLoadedMsg struct{ inbounds []*store.Inbound }
type inboundCreatedMsg struct{ inbound *store.Inbound }
type inboundDeletedMsg struct{}
type inboundUpdatedMsg struct{}
type statsLoadedMsg struct{ stats []store.TrafficSummary }
type daemonStatusMsg struct{ running bool }

// Commands

func loadInbounds(st *store.Store) tea.Cmd {
	return func() tea.Msg {
		list, err := st.ListInbounds()
		if err != nil {
			return errMsg{err}
		}
		return inboundsLoadedMsg{list}
	}
}

func loadStats(st *store.Store) tea.Cmd {
	return func() tea.Msg {
		stats, err := st.GetTrafficSummary()
		if err != nil {
			return errMsg{err}
		}
		return statsLoadedMsg{stats}
	}
}

func checkDaemon(dataDir string) tea.Cmd {
	return func() tea.Msg {
		running := isDaemonRunning(dataDir)
		return daemonStatusMsg{running}
	}
}

// reloadDaemon 通过 PID 文件向守护进程发送 SIGHUP 信号。
func reloadDaemon(dataDir string) {
	pidFile := filepath.Join(dataDir, "sing-box.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = process.Signal(syscall.SIGHUP)
}

func isDaemonRunning(dataDir string) bool {
	pidFile := filepath.Join(dataDir, "sing-box.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// 在 Unix 上 FindProcess 总是成功；用 signal(0) 检测进程是否存在。
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func generateShareURL(ib *store.Inbound, host string) string {
	p := protocol.Get(ib.Protocol)
	if p == nil {
		return ""
	}
	return p.GenerateURL(ib, host)
}

func extractHost() string {
	// TUI 独立模式下没有监听地址，使用占位符，用户需替换为服务器 IP。
	return "YOUR_SERVER_IP"
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
