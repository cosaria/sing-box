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

func (a app) viewAdd() string {
	return menuStyle.Render("添加配置（开发中）\n\nEsc 返回")
}
