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
			a.list.editMode = false
		}
	}
	return a, nil
}

func (a app) viewList() string {
	return menuStyle.Render("配置列表（开发中）\n\nEsc 返回")
}

func (a app) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "esc" {
		a.state = stateList
	}
	return a, nil
}

func (a app) viewDetail() string {
	return menuStyle.Render("配置详情（开发中）\n\nEsc 返回")
}
