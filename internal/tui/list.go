package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cosaria/sing-box/internal/store"
)

type listModel struct {
	inbounds []*store.Inbound
	cursor   int
	editMode bool
}

func newListModel() listModel { return listModel{} }

type detailModel struct {
	inbound  *store.Inbound
	shareURL string
	confirm  bool
}

func deleteInboundCmd(st *store.Store, dataDir string, id int64) tea.Cmd {
	return func() tea.Msg {
		if err := st.DeleteInbound(id); err != nil {
			return errMsg{err}
		}
		reloadDaemon(dataDir)
		return inboundDeletedMsg{}
	}
}

func (a app) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case inboundsLoadedMsg:
		a.list.inbounds = msg.inbounds
		if a.list.cursor >= len(msg.inbounds) && len(msg.inbounds) > 0 {
			a.list.cursor = len(msg.inbounds) - 1
		}
		return a, nil
	case inboundDeletedMsg:
		return a, loadInbounds(a.store)
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			a.state = stateMenu
			a.list.editMode = false
			return a, nil
		case "up", "k":
			if a.list.cursor > 0 {
				a.list.cursor--
			}
		case "down", "j":
			if a.list.cursor < len(a.list.inbounds)-1 {
				a.list.cursor++
			}
		case "r":
			return a, loadInbounds(a.store)
		case "enter":
			if len(a.list.inbounds) == 0 {
				return a, nil
			}
			selected := a.list.inbounds[a.list.cursor]
			if a.list.editMode {
				a.state = stateEdit
				a.edit = newEditModel(selected)
				return a, nil
			}
			host := extractHost()
			shareURL := generateShareURL(selected, host)
			a.detail = detailModel{
				inbound:  selected,
				shareURL: shareURL,
				confirm:  false,
			}
			a.state = stateDetail
			return a, nil
		}
	}
	return a, nil
}

func (a app) viewList() string {
	title := "入站配置列表"
	if a.list.editMode {
		title = "选择要编辑的配置"
	}
	content := titleStyle.Render(title) + "\n\n"

	if len(a.list.inbounds) == 0 {
		content += dimStyle.Render("暂无配置") + "\n"
	} else {
		for i, ib := range a.list.inbounds {
			cursor := "  "
			style := normalStyle
			tagStyle := dimStyle
			if a.list.cursor == i {
				cursor = "> "
				style = selectedStyle
				tagStyle = infoStyle
			}
			line := fmt.Sprintf("%-12s  %-6d  %s",
				style.Render(ib.Protocol),
				ib.Port,
				tagStyle.Render(ib.Tag),
			)
			content += cursor + line + "\n"
		}
	}

	hints := "↑↓ 选择  r 刷新  Esc 返回"
	if len(a.list.inbounds) > 0 {
		if a.list.editMode {
			hints = "↑↓ 选择  Enter 编辑  r 刷新  Esc 返回"
		} else {
			hints = "↑↓ 选择  Enter 查看  r 刷新  Esc 返回"
		}
	}
	content += "\n" + dimStyle.Render(hints)
	return menuStyle.Render(content)
}

func (a app) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case inboundDeletedMsg:
		a.state = stateList
		a.detail = detailModel{}
		return a, loadInbounds(a.store)
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if a.detail.confirm {
			switch msg.String() {
			case "y", "Y":
				if a.detail.inbound != nil {
					return a, deleteInboundCmd(a.store, a.dataDir, a.detail.inbound.ID)
				}
			case "n", "N", "esc":
				a.detail.confirm = false
			}
			return a, nil
		}
		switch msg.String() {
		case "esc", "q":
			a.state = stateList
			a.detail = detailModel{}
		case "d":
			a.detail.confirm = true
		case "e":
			if a.detail.inbound != nil {
				a.state = stateEdit
				a.edit = newEditModel(a.detail.inbound)
			}
		}
	}
	return a, nil
}

func (a app) viewDetail() string {
	if a.detail.inbound == nil {
		return menuStyle.Render(dimStyle.Render("无数据") + "\n\n" + dimStyle.Render("Esc 返回"))
	}
	ib := a.detail.inbound

	content := titleStyle.Render("配置详情") + "\n\n"
	content += infoStyle.Render("Tag：    ") + normalStyle.Render(ib.Tag) + "\n"
	content += infoStyle.Render("协议：   ") + normalStyle.Render(ib.Protocol) + "\n"
	content += infoStyle.Render("端口：   ") + normalStyle.Render(fmt.Sprintf("%d", ib.Port)) + "\n"

	if a.detail.shareURL != "" {
		content += "\n" + infoStyle.Render("分享链接：") + "\n"
		content += dimStyle.Render(a.detail.shareURL) + "\n"
		qr := RenderQR(a.detail.shareURL)
		if qr != "" {
			content += "\n" + qr
		}
	}

	if a.detail.confirm {
		content += "\n" + errorStyle.Render("确认删除？(y/n)")
	} else {
		hints := []string{"d 删除", "e 编辑", "Esc 返回"}
		content += "\n" + dimStyle.Render(strings.Join(hints, "  "))
	}

	return menuStyle.Render(content)
}
