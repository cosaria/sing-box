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

func (a app) viewStats() string {
	return menuStyle.Render("流量统计（开发中）\n\nEsc 返回")
}
