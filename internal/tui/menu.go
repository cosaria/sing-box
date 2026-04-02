package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

var menuItems = []string{
	"添加配置", "查看配置", "修改配置", "流量统计", "服务状态", "退出",
}

type menuModel struct{ cursor int }

func newMenuModel() menuModel { return menuModel{} }

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
	case 5:
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
