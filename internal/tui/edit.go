package tui

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cosaria/sing-box/internal/store"
)

type editModel struct {
	inbound   *store.Inbound
	portInput textinput.Model
	focused   bool
	err       error
}

func newEditModel(ib *store.Inbound) editModel {
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

func updateInbound(st *store.Store, dataDir string, ib *store.Inbound, port uint16) tea.Cmd {
	return func() tea.Msg {
		ib.Port = port
		if err := st.UpdateInbound(ib); err != nil {
			return errMsg{err}
		}
		reloadDaemon(dataDir)
		return inboundUpdatedMsg{}
	}
}

func (a app) updateEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case inboundUpdatedMsg:
		a.state = stateList
		tag := ""
		port := uint16(0)
		if a.edit.inbound != nil {
			tag = a.edit.inbound.Tag
			port = a.edit.inbound.Port
		}
		a.edit = editModel{}
		a.message = fmt.Sprintf("已更新 %s 端口为 %d", tag, port)
		return a, loadInbounds(a.store)
	}
	switch msg := msg.(type) {
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
			return a, updateInbound(a.store, a.dataDir, a.edit.inbound, port)
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
