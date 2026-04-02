package tui

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cosaria/sing-box/internal/store"
)

var protocols = []string{"shadowsocks", "vless", "trojan"}
var protocolNames = []string{"Shadowsocks (SS2022)", "VLESS-REALITY", "Trojan"}

type addModel struct {
	step      int
	protocol  int
	portInput textinput.Model
	result    *store.Inbound
	err       error
}

func newAddModel() addModel {
	ti := textinput.New()
	ti.Placeholder = "输入端口（1-65535）"
	ti.CharLimit = 5
	ti.Width = 20
	return addModel{
		step:      0,
		protocol:  0,
		portInput: ti,
	}
}

func (m addModel) init() tea.Cmd { return nil }

func createInbound(st *store.Store, protocol string, port uint16) tea.Cmd {
	return func() tea.Msg {
		ib := &store.Inbound{
			Protocol: protocol,
			Port:     port,
			Settings: "{}",
		}
		// 生成唯一 Tag
		ib.Tag = fmt.Sprintf("%s-%d", protocol, port)
		if err := st.CreateInbound(ib); err != nil {
			return errMsg{err}
		}
		return inboundCreatedMsg{ib}
	}
}

func (a app) updateAdd(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case inboundCreatedMsg:
		a.add.result = msg.inbound
		a.add.step = 3
		reloadDaemon(a.dataDir)
		return a, nil
	case errMsg:
		a.add.err = msg.err
		a.add.step = 3
		return a, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if a.add.step == 0 {
				a.state = stateMenu
				return a, nil
			}
			a.add.step--
			if a.add.step == 1 {
				a.add.portInput.Blur()
			}
			return a, nil
		case "ctrl+c":
			return a, tea.Quit
		}

		switch a.add.step {
		case 0: // 协议选择
			switch msg.String() {
			case "up", "k":
				if a.add.protocol > 0 {
					a.add.protocol--
				}
			case "down", "j":
				if a.add.protocol < len(protocols)-1 {
					a.add.protocol++
				}
			case "enter":
				a.add.step = 1
				a.add.portInput.Focus()
			}
		case 1: // 端口输入
			switch msg.String() {
			case "enter":
				portStr := a.add.portInput.Value()
				portNum, err := strconv.ParseUint(portStr, 10, 16)
				if err != nil || portNum < 1 || portNum > 65535 {
					a.add.err = fmt.Errorf("端口必须是 1-65535 之间的数字")
					return a, nil
				}
				a.add.err = nil
				a.add.step = 2
				a.add.portInput.Blur()
			default:
				var cmd tea.Cmd
				a.add.portInput, cmd = a.add.portInput.Update(msg)
				return a, cmd
			}
		case 2: // 确认
			if msg.String() == "enter" {
				portStr := a.add.portInput.Value()
				portNum, _ := strconv.ParseUint(portStr, 10, 16)
				return a, createInbound(a.store, protocols[a.add.protocol], uint16(portNum))
			}
		case 3: // 结果
			a.state = stateMenu
			a.add = newAddModel()
			return a, nil
		}
	}

	if a.add.step == 1 {
		var cmd tea.Cmd
		a.add.portInput, cmd = a.add.portInput.Update(msg)
		return a, cmd
	}

	return a, nil
}

func (a app) viewAdd() string {
	var content string

	switch a.add.step {
	case 0:
		content = titleStyle.Render("添加入站配置") + "\n\n"
		content += dimStyle.Render("选择协议：") + "\n\n"
		for i, name := range protocolNames {
			cursor := "  "
			style := normalStyle
			if a.add.protocol == i {
				cursor = "> "
				style = selectedStyle
			}
			content += fmt.Sprintf("%s%s\n", cursor, style.Render(name))
		}
		content += "\n" + dimStyle.Render("↑↓ 选择  Enter 确认  Esc 返回")

	case 1:
		content = titleStyle.Render("添加入站配置") + "\n\n"
		content += infoStyle.Render("协议："+protocolNames[a.add.protocol]) + "\n\n"
		content += dimStyle.Render("输入监听端口：") + "\n"
		content += a.add.portInput.View() + "\n"
		if a.add.err != nil {
			content += "\n" + errorStyle.Render(a.add.err.Error())
		}
		content += "\n" + dimStyle.Render("Enter 下一步  Esc 返回")

	case 2:
		content = titleStyle.Render("确认创建") + "\n\n"
		content += infoStyle.Render("协议：  ") + normalStyle.Render(protocolNames[a.add.protocol]) + "\n"
		content += infoStyle.Render("端口：  ") + normalStyle.Render(a.add.portInput.Value()) + "\n"
		content += "\n" + dimStyle.Render("Enter 确认创建  Esc 返回")

	case 3:
		if a.add.err != nil {
			content = titleStyle.Render("创建失败") + "\n\n"
			content += errorStyle.Render(a.add.err.Error()) + "\n"
		} else if a.add.result != nil {
			host := extractHost()
			shareURL := generateShareURL(a.add.result, host)
			content = titleStyle.Render("创建成功！") + "\n\n"
			content += infoStyle.Render("Tag：  ") + normalStyle.Render(a.add.result.Tag) + "\n"
			content += infoStyle.Render("协议：  ") + normalStyle.Render(a.add.result.Protocol) + "\n"
			content += infoStyle.Render("端口：  ") + normalStyle.Render(fmt.Sprintf("%d", a.add.result.Port)) + "\n"
			if shareURL != "" {
				content += "\n" + infoStyle.Render("分享链接：") + "\n"
				content += dimStyle.Render(shareURL) + "\n"
				qr := RenderQR(shareURL)
				if qr != "" {
					content += "\n" + qr
				}
			}
		}
		content += "\n" + dimStyle.Render("任意键返回主菜单")
	}

	return menuStyle.Render(content)
}
