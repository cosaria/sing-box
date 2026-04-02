package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

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
	content := titleStyle.Render("流量统计") + "\n\n"

	if len(a.stats.stats) == 0 {
		content += dimStyle.Render("暂无流量数据") + "\n"
	} else {
		// 表头
		header := fmt.Sprintf("%-20s  %-12s  %-12s",
			infoStyle.Render("Tag"),
			infoStyle.Render("上传"),
			infoStyle.Render("下载"),
		)
		content += header + "\n"
		content += dimStyle.Render("────────────────────  ────────────  ────────────") + "\n"

		for _, s := range a.stats.stats {
			row := fmt.Sprintf("%-20s  %-12s  %-12s",
				normalStyle.Render(s.Tag),
				successStyle.Render(FormatBytes(s.Upload)),
				infoStyle.Render(FormatBytes(s.Download)),
			)
			content += row + "\n"
		}
	}

	content += "\n" + dimStyle.Render("r 刷新  Esc 返回")
	return menuStyle.Render(content)
}
