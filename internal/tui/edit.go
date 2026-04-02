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

func (a app) viewEdit() string {
	return menuStyle.Render("修改配置（开发中）\n\nEsc 返回")
}
