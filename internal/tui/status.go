package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

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
	content := titleStyle.Render("服务状态") + "\n\n"

	if a.status.status == nil {
		content += dimStyle.Render("加载中...") + "\n"
	} else {
		s := a.status.status

		// 引擎状态
		engineLabel := infoStyle.Render("引擎状态：  ")
		var engineVal string
		if s.Engine == "running" {
			engineVal = successStyle.Render("运行中 ●")
		} else {
			engineVal = errorStyle.Render("已停止 ○")
		}
		content += engineLabel + engineVal + "\n"

		// 入站数量
		content += infoStyle.Render("入站数量：  ") + normalStyle.Render(fmt.Sprintf("%d", s.Inbounds)) + "\n"

		// API 地址
		content += infoStyle.Render("API 地址：  ") + dimStyle.Render("http://"+a.listenAddr) + "\n"

		// 订阅地址
		if a.subToken != "" {
			subURL := fmt.Sprintf("http://%s/sub/%s", a.listenAddr, a.subToken)
			content += "\n" + infoStyle.Render("订阅地址：") + "\n"
			content += dimStyle.Render(subURL) + "\n"
			qr := RenderQR(subURL)
			if qr != "" {
				content += "\n" + qr
			}
		}
	}

	content += "\n" + dimStyle.Render("r 刷新  Esc 返回")
	return menuStyle.Render(content)
}
