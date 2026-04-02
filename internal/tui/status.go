package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type statusModel struct {
	running       bool
	inboundCount  int
	loaded        bool
}

func newStatusModel() statusModel { return statusModel{} }

// statusLoadedMsg 携带守护进程状态和 inbound 数量。
type statusLoadedMsg struct {
	running      bool
	inboundCount int
}

// checkDaemonFull 同时检查守护进程状态和 inbound 数量。
func checkDaemonFull(a app) tea.Cmd {
	return func() tea.Msg {
		running := isDaemonRunning(a.dataDir)
		inbounds, err := a.store.ListInbounds()
		count := 0
		if err == nil {
			count = len(inbounds)
		}
		return statusLoadedMsg{running: running, inboundCount: count}
	}
}

func (a app) updateStatus(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case daemonStatusMsg:
		// 兼容旧式消息（仅守护进程状态）
		a.status.running = msg.running
		a.status.loaded = true
	case statusLoadedMsg:
		a.status.running = msg.running
		a.status.inboundCount = msg.inboundCount
		a.status.loaded = true
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			a.state = stateMenu
		case "r":
			return a, checkDaemonFull(a)
		}
	}
	return a, nil
}

func (a app) viewStatus() string {
	content := titleStyle.Render("服务状态") + "\n\n"

	if !a.status.loaded {
		content += dimStyle.Render("加载中...") + "\n"
	} else {
		// 守护进程状态
		engineLabel := infoStyle.Render("守护进程：  ")
		var engineVal string
		if a.status.running {
			engineVal = successStyle.Render("运行中 ●")
		} else {
			engineVal = errorStyle.Render("已停止 ○")
		}
		content += engineLabel + engineVal + "\n"

		// inbound 数量
		content += infoStyle.Render("入站配置：  ") +
			normalStyle.Render(fmt.Sprintf("%d 条", a.status.inboundCount)) + "\n"
	}

	content += "\n" + dimStyle.Render("r 刷新  Esc 返回")
	return menuStyle.Render(content)
}
