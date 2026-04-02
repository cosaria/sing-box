# sing-box panel

sing-box 代理管理面板 — 单二进制 TUI 工具，嵌入 sing-box 核心，支持多协议管理。

## 特性

- **单二进制** — Go 编译，嵌入 sing-box 核心，零外部依赖
- **TUI 管理** — Bubbletea 终端界面，交互式管理代理配置
- **多协议** — Shadowsocks (SS2022)、VLESS-REALITY、Trojan
- **自动配置** — 创建 inbound 时自动生成密码/密钥对/UUID
- **分享链接 + QR** — 一键显示 ss://、vless://、trojan:// 链接和二维码
- **流量统计** — 实时 per-inbound 上传/下载统计
- **自更新** — `sing-box update` 从 GitHub Releases 原子替换
- **服务管理** — systemd + OpenRC 一键安装/卸载
- **崩溃恢复** — 引擎 panic 自动恢复，60 秒内 3 次崩溃触发熔断

## 安装

```bash
bash <(curl -sL https://raw.githubusercontent.com/cosaria/sing-box/main/install.sh)
```

选项:
- `-l` 本地编译安装
- `-v v0.1.0` 指定版本
- `-p http://proxy:port` 使用代理下载

## 使用

```bash
sing-box              # 打开 TUI 管理面板
sing-box serve        # 守护进程模式（systemd 使用）
sing-box update       # 检查并更新到最新版本
sing-box version      # 显示版本信息
```

### TUI 菜单

```
╭─ sing-box 管理面板 ─────────────────╮
│  > 添加配置                          │
│    查看配置                          │
│    修改配置                          │
│    流量统计                          │
│    服务状态                          │
│    退出                             │
╰─────────────────────────────────────╯
```

## 架构

```
sing-box serve (守护进程)              sing-box (TUI)
├── sing-box 引擎                     ├── SQLite 直接读写
├── 流量统计采集                       ├── 配置增删改查
├── PID 文件                          ├── 分享链接 + QR 码
└── SIGHUP 重载                       └── SIGHUP 通知守护进程
```

TUI 和守护进程通过 SQLite（WAL 模式）+ Unix 信号协调。无 HTTP API。

## 支持的协议

| 协议 | 分享链接 | 自动生成 |
|------|---------|---------|
| Shadowsocks SS2022 | `ss://...` | method + UUID 密码 |
| VLESS-REALITY | `vless://...` | UUID + X25519 密钥对 + short_id |
| Trojan | `trojan://...` | UUID 密码 |

## 开发

```bash
go test ./... -v -count=1 -race    # 运行测试
go build -o sing-box ./cmd/sing-box/  # 编译
bash install.sh -l                    # 本地安装
```

## License

MIT
