package tui

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cosaria/sing-box/internal/protocol"
	"github.com/cosaria/sing-box/internal/store"
)

type addModel struct {
	step      int
	protocols []protocol.Protocol // 从注册表动态加载
	protoIdx  int
	portInput textinput.Model
	result    *store.Inbound
	err       error
}

func newAddModel() addModel {
	ti := textinput.New()
	ti.Placeholder = "输入端口（1-65535）"
	ti.CharLimit = 5
	ti.Width = 20

	// 动态获取已注册协议并按名称排序
	protos := protocol.All()
	sort.Slice(protos, func(i, j int) bool {
		return protos[i].Name() < protos[j].Name()
	})

	return addModel{
		step:      0,
		protocols: protos,
		protoIdx:  0,
		portInput: ti,
	}
}

func (m addModel) init() tea.Cmd { return nil }

// createInbound 调用协议注册表生成默认配置并写入 store。
func createInbound(st *store.Store, dataDir string, proto string, port uint16) tea.Cmd {
	return func() tea.Msg {
		p := protocol.Get(proto)
		if p == nil {
			return errMsg{fmt.Errorf("不支持的协议: %s", proto)}
		}
		settings, err := p.DefaultSettings(port)
		if err != nil {
			return errMsg{fmt.Errorf("生成默认配置失败: %w", err)}
		}
		tag := fmt.Sprintf("%s-%d", proto, port)
		ib := &store.Inbound{
			Tag:      tag,
			Protocol: proto,
			Port:     port,
			Settings: settings,
		}
		if err := st.CreateInbound(ib); err != nil {
			return errMsg{err}
		}
		reloadDaemon(dataDir)
		return inboundCreatedMsg{ib}
	}
}

func (a app) updateAdd(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case inboundCreatedMsg:
		a.add.result = msg.inbound
		a.add.step = 3
		a.add.err = nil
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
				if a.add.protoIdx > 0 {
					a.add.protoIdx--
				}
			case "down", "j":
				if a.add.protoIdx < len(a.add.protocols)-1 {
					a.add.protoIdx++
				}
			case "enter":
				if len(a.add.protocols) > 0 {
					a.add.step = 1
					a.add.portInput.Focus()
				}
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
				selectedProto := a.add.protocols[a.add.protoIdx].Name()
				return a, createInbound(a.store, a.dataDir, selectedProto, uint16(portNum))
			}

		case 3: // 结果：任意键返回
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
	case 0: // 协议选择
		content = titleStyle.Render("添加入站配置") + "\n\n"
		if len(a.add.protocols) == 0 {
			content += errorStyle.Render("无可用协议，请先注册协议。") + "\n"
		} else {
			content += dimStyle.Render("选择协议：") + "\n\n"
			for i, p := range a.add.protocols {
				cursor := "  "
				style := normalStyle
				if a.add.protoIdx == i {
					cursor = "> "
					style = selectedStyle
				}
				content += fmt.Sprintf("%s%s\n", cursor, style.Render(p.DisplayName()))
			}
		}
		content += "\n" + dimStyle.Render("↑↓ 选择  Enter 确认  Esc 返回")

	case 1: // 端口输入
		protoDisplay := ""
		if len(a.add.protocols) > 0 {
			protoDisplay = a.add.protocols[a.add.protoIdx].DisplayName()
		}
		content = titleStyle.Render("添加入站配置") + "\n\n"
		content += infoStyle.Render("协议：") + normalStyle.Render(protoDisplay) + "\n\n"
		content += dimStyle.Render("输入监听端口：") + "\n"
		content += a.add.portInput.View() + "\n"
		if a.add.err != nil {
			content += "\n" + errorStyle.Render(a.add.err.Error())
		}
		content += "\n" + dimStyle.Render("Enter 下一步  Esc 返回")

	case 2: // 确认
		protoDisplay := ""
		if len(a.add.protocols) > 0 {
			protoDisplay = a.add.protocols[a.add.protoIdx].DisplayName()
		}
		content = titleStyle.Render("确认创建") + "\n\n"
		content += infoStyle.Render("协议：  ") + normalStyle.Render(protoDisplay) + "\n"
		content += infoStyle.Render("端口：  ") + normalStyle.Render(a.add.portInput.Value()) + "\n"
		content += "\n" + dimStyle.Render("Enter 确认创建  Esc 返回")

	case 3: // 结果
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
