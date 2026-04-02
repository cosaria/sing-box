package tui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cosaria/sing-box/internal/protocol"
	"github.com/cosaria/sing-box/internal/store"
)

// SS 可选加密方式
var ssMethods = []string{
	"2022-blake3-aes-128-gcm",
	"2022-blake3-aes-256-gcm",
	"2022-blake3-chacha20-poly1305",
	"aes-256-gcm",
	"aes-128-gcm",
	"chacha20-ietf-poly1305",
	"xchacha20-ietf-poly1305",
}

type addModel struct {
	step      int // 0=协议, 1=SS加密方式(仅SS), 2=端口, 3=确认, 4=结果
	protocols []protocol.Protocol
	protoIdx  int
	methodIdx int // SS 加密方式选择
	portInput textinput.Model
	result    *store.Inbound
	err       error
}

func newAddModel() addModel {
	ti := textinput.New()
	ti.Placeholder = "输入端口（1-65535）"
	ti.CharLimit = 5
	ti.Width = 20

	protos := protocol.All()
	sort.Slice(protos, func(i, j int) bool {
		return protos[i].Name() < protos[j].Name()
	})

	return addModel{
		protocols: protos,
		portInput: ti,
	}
}

func (m addModel) init() tea.Cmd { return nil }

func (m addModel) selectedProtoName() string {
	if len(m.protocols) == 0 {
		return ""
	}
	return m.protocols[m.protoIdx].Name()
}

func (m addModel) isSS() bool {
	return m.selectedProtoName() == "shadowsocks"
}

func createInbound(st *store.Store, dataDir string, proto string, port uint16, customSettings string) tea.Cmd {
	return func() tea.Msg {
		p := protocol.Get(proto)
		if p == nil {
			return errMsg{fmt.Errorf("不支持的协议: %s", proto)}
		}

		var settings string
		if customSettings != "" {
			settings = customSettings
		} else {
			var err error
			settings, err = p.DefaultSettings(port)
			if err != nil {
				return errMsg{fmt.Errorf("生成默认配置失败: %w", err)}
			}
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

// generateSSSettings 生成指定加密方式的 SS 配置
func generateSSSettings(method string) string {
	uuid := protocol.GenerateUUID()
	b, _ := json.Marshal(map[string]string{
		"method":   method,
		"password": uuid,
	})
	return string(b)
}

func (a app) updateAdd(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case inboundCreatedMsg:
		a.add.result = msg.inbound
		a.add.step = 4
		a.add.err = nil
		return a, nil
	case errMsg:
		a.add.err = msg.err
		a.add.step = 4
		return a, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if a.add.step == 0 {
				a.state = stateMenu
				return a, nil
			}
			// SS: 从端口(2)返回到加密方式(1)
			// 非SS: 从端口(2)返回到协议(0)
			if a.add.step == 2 && !a.add.isSS() {
				a.add.step = 0
			} else {
				a.add.step--
			}
			a.add.portInput.Blur()
			a.add.err = nil
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
					if a.add.isSS() {
						a.add.step = 1 // SS: 进入加密方式选择
					} else {
						a.add.step = 2 // 其他: 直接进入端口
						a.add.portInput.Focus()
					}
				}
			}

		case 1: // SS 加密方式选择
			switch msg.String() {
			case "up", "k":
				if a.add.methodIdx > 0 {
					a.add.methodIdx--
				}
			case "down", "j":
				if a.add.methodIdx < len(ssMethods)-1 {
					a.add.methodIdx++
				}
			case "enter":
				a.add.step = 2
				a.add.portInput.Focus()
			}

		case 2: // 端口输入
			switch msg.String() {
			case "enter":
				portStr := a.add.portInput.Value()
				portNum, err := strconv.ParseUint(portStr, 10, 16)
				if err != nil || portNum < 1 || portNum > 65535 {
					a.add.err = fmt.Errorf("端口必须是 1-65535 之间的数字")
					return a, nil
				}
				a.add.err = nil
				a.add.step = 3
				a.add.portInput.Blur()
			default:
				var cmd tea.Cmd
				a.add.portInput, cmd = a.add.portInput.Update(msg)
				return a, cmd
			}

		case 3: // 确认
			if msg.String() == "enter" {
				portStr := a.add.portInput.Value()
				portNum, _ := strconv.ParseUint(portStr, 10, 16)
				protoName := a.add.selectedProtoName()
				var customSettings string
				if a.add.isSS() {
					customSettings = generateSSSettings(ssMethods[a.add.methodIdx])
				}
				return a, createInbound(a.store, a.dataDir, protoName, uint16(portNum), customSettings)
			}

		case 4: // 结果
			a.state = stateMenu
			a.add = newAddModel()
			return a, nil
		}
	}

	if a.add.step == 2 {
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
			content += errorStyle.Render("无可用协议") + "\n"
		} else {
			content += "选择协议：\n\n"
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

	case 1: // SS 加密方式
		content = titleStyle.Render("添加入站配置") + "\n\n"
		content += infoStyle.Render("协议：") + " Shadowsocks\n\n"
		content += "选择加密方式：\n\n"
		for i, m := range ssMethods {
			cursor := "  "
			style := normalStyle
			if a.add.methodIdx == i {
				cursor = "> "
				style = selectedStyle
			}
			content += fmt.Sprintf("%s%s\n", cursor, style.Render(m))
		}
		content += "\n" + dimStyle.Render("↑↓ 选择  Enter 确认  Esc 返回")

	case 2: // 端口输入
		protoDisplay := ""
		if len(a.add.protocols) > 0 {
			protoDisplay = a.add.protocols[a.add.protoIdx].DisplayName()
		}
		content = titleStyle.Render("添加入站配置") + "\n\n"
		content += infoStyle.Render("协议：") + " " + protoDisplay + "\n"
		if a.add.isSS() {
			content += infoStyle.Render("加密：") + " " + ssMethods[a.add.methodIdx] + "\n"
		}
		content += "\n输入监听端口：\n"
		content += a.add.portInput.View() + "\n"
		if a.add.err != nil {
			content += "\n" + errorStyle.Render(a.add.err.Error())
		}
		content += "\n" + dimStyle.Render("Enter 下一步  Esc 返回")

	case 3: // 确认
		protoDisplay := ""
		if len(a.add.protocols) > 0 {
			protoDisplay = a.add.protocols[a.add.protoIdx].DisplayName()
		}
		content = titleStyle.Render("确认创建") + "\n\n"
		content += infoStyle.Render("协议：") + "  " + protoDisplay + "\n"
		if a.add.isSS() {
			content += infoStyle.Render("加密：") + "  " + ssMethods[a.add.methodIdx] + "\n"
		}
		content += infoStyle.Render("端口：") + "  " + a.add.portInput.Value() + "\n"
		content += "\n" + selectedStyle.Render("按 Enter 确认创建") + "\n"
		content += "\n" + dimStyle.Render("Enter 创建  Esc 返回")

	case 4: // 结果
		if a.add.err != nil {
			content = titleStyle.Render("创建失败") + "\n\n"
			content += errorStyle.Render(a.add.err.Error()) + "\n"
		} else if a.add.result != nil {
			host := extractHost()
			shareURL := generateShareURL(a.add.result, host)
			content = successStyle.Render("✓ 创建成功") + "\n\n"
			content += infoStyle.Render("Tag：") + "  " + a.add.result.Tag + "\n"
			content += infoStyle.Render("端口：") + " " + fmt.Sprintf("%d", a.add.result.Port) + "\n"
			if shareURL != "" {
				content += "\n" + infoStyle.Render("分享链接：") + "\n"
				content += shareURL + "\n"
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
