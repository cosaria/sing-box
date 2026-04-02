package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type statusModel struct {
	running bool
	loaded  bool
}

func newStatusModel() statusModel { return statusModel{} }

func (a app) updateStatus(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case daemonStatusMsg:
		a.status.running = msg.running
		a.status.loaded = true
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			a.state = stateMenu
		case "r":
			return a, checkDaemon(a.dataDir)
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
	}

	content += "\n" + dimStyle.Render("r 刷新  Esc 返回")
	return menuStyle.Render(content)
}
