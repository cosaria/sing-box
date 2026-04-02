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

func (a app) viewStatus() string {
	return menuStyle.Render("服务状态（开发中）\n\nEsc 返回")
}
