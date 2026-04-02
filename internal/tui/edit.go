package tui

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type editModel struct {
	inbound   *Inbound
	portInput textinput.Model
	focused   bool
	err       error
}

func newEditModel(ib *Inbound) editModel {
	ti := textinput.New()
	ti.Placeholder = "输入新端口（1-65535）"
	ti.CharLimit = 5
	ti.Width = 20
	ti.SetValue(fmt.Sprintf("%d", ib.Port))
	ti.Focus()
	return editModel{
		inbound:   ib,
		portInput: ti,
		focused:   true,
	}
}

func updateInbound(c *Client, id int64, port *uint16) tea.Cmd {
	return func() tea.Msg {
		ib, err := c.UpdateInbound(id, port, nil)
		if err != nil {
			return errMsg{err}
		}
		return inboundUpdatedMsg{ib}
	}
}

func (a app) updateEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case inboundUpdatedMsg:
		a.state = stateList
		a.edit = editModel{}
		a.message = fmt.Sprintf("已更新 %s 端口为 %d", msg.inbound.Tag, msg.inbound.Port)
		return a, loadInbounds(a.client)
	case errMsg:
		a.edit.err = msg.err
		return a, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			a.state = stateList
			a.edit = editModel{}
			return a, nil
		case "enter":
			portStr := a.edit.portInput.Value()
			portNum, err := strconv.ParseUint(portStr, 10, 16)
			if err != nil || portNum < 1 || portNum > 65535 {
				a.edit.err = fmt.Errorf("端口必须是 1-65535 之间的数字")
				return a, nil
			}
			a.edit.err = nil
			port := uint16(portNum)
			return a, updateInbound(a.client, a.edit.inbound.ID, &port)
		default:
			var cmd tea.Cmd
			a.edit.portInput, cmd = a.edit.portInput.Update(msg)
			return a, cmd
		}
	}

	var cmd tea.Cmd
	a.edit.portInput, cmd = a.edit.portInput.Update(msg)
	return a, cmd
}

func (a app) viewEdit() string {
	if a.edit.inbound == nil {
		return menuStyle.Render(dimStyle.Render("无数据") + "\n\n" + dimStyle.Render("Esc 返回"))
	}

	content := titleStyle.Render("修改配置") + "\n\n"
	content += infoStyle.Render("Tag：    ") + normalStyle.Render(a.edit.inbound.Tag) + "\n"
	content += infoStyle.Render("协议：   ") + normalStyle.Render(a.edit.inbound.Protocol) + "\n"
	content += "\n"
	content += dimStyle.Render("修改端口：") + "\n"
	content += a.edit.portInput.View() + "\n"

	if a.edit.err != nil {
		content += "\n" + errorStyle.Render(a.edit.err.Error())
	}

	content += "\n" + dimStyle.Render("Enter 保存  Esc 取消")
	return menuStyle.Render(content)
}
