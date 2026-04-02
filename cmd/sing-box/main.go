package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/cosaria/sing-box/internal/api"
	"github.com/cosaria/sing-box/internal/engine"
	"github.com/cosaria/sing-box/internal/platform"
	"github.com/cosaria/sing-box/internal/service"
	"github.com/cosaria/sing-box/internal/stats"
	"github.com/cosaria/sing-box/internal/store"
	"github.com/cosaria/sing-box/internal/tui"
	"github.com/cosaria/sing-box/internal/updater"
	"github.com/spf13/cobra"

	_ "github.com/cosaria/sing-box/internal/protocol"
)

// Version 由构建时 ldflags 注入，默认为 "dev"。
var Version = "dev"

func main() {
	plat := platform.Detect()

	rootCmd := &cobra.Command{
		Use:   "sing-box",
		Short: "sing-box 管理面板",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, _ := cmd.Flags().GetString("listen")
			dataDir, _ := cmd.Flags().GetString("data-dir")
			return runTUI(listenAddr, dataDir)
		},
	}
	rootCmd.Flags().String("listen", "127.0.0.1:9090", "API 监听地址")
	rootCmd.Flags().String("data-dir", plat.DataDir, "数据目录")

	rootCmd.AddCommand(serveCmd())
	rootCmd.AddCommand(updateCmd())
	rootCmd.AddCommand(serviceCmd())
	rootCmd.AddCommand(versionCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// serveCmd 启动守护进程（HTTP API + sing-box 引擎），不启动 TUI。
func serveCmd() *cobra.Command {
	var (
		listenAddr string
		dataDir    string
	)
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "启动守护进程（HTTP API + sing-box 引擎）",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(listenAddr, dataDir)
		},
	}
	plat := platform.Detect()
	cmd.Flags().StringVar(&listenAddr, "listen", "127.0.0.1:9090", "API 监听地址")
	cmd.Flags().StringVar(&dataDir, "data-dir", plat.DataDir, "数据目录")
	return cmd
}

// updateCmd 检查并下载最新版本，原子替换当前二进制。
func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "自动更新到最新版本",
		RunE: func(cmd *cobra.Command, args []string) error {
			return updater.Update(Version)
		},
	}
}

// serviceCmd 管理系统服务（安装 / 卸载）。
func serviceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "管理系统服务",
	}
	cmd.AddCommand(serviceInstallCmd())
	cmd.AddCommand(serviceUninstallCmd())
	return cmd
}

func serviceInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "安装系统服务（systemd / openrc）",
		RunE: func(cmd *cobra.Command, args []string) error {
			plat := platform.Detect()
			mgr := service.NewManager(plat.InitSystem)
			if mgr == nil {
				return fmt.Errorf("不支持的 init 系统: %s", plat.InitSystem)
			}
			if err := mgr.Install(plat.BinPath, plat.DataDir); err != nil {
				return fmt.Errorf("安装服务失败: %w", err)
			}
			fmt.Println("服务安装成功")
			return nil
		},
	}
}

func serviceUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "卸载系统服务",
		RunE: func(cmd *cobra.Command, args []string) error {
			plat := platform.Detect()
			mgr := service.NewManager(plat.InitSystem)
			if mgr == nil {
				return fmt.Errorf("不支持的 init 系统: %s", plat.InitSystem)
			}
			if err := mgr.Uninstall(); err != nil {
				return fmt.Errorf("卸载服务失败: %w", err)
			}
			fmt.Println("服务卸载成功")
			return nil
		},
	}
}

// versionCmd 打印版本信息。
func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("sing-box panel %s (%s %s/%s)\n",
				Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		},
	}
}

// runTUI 启动后端组件并进入 TUI 交互界面，退出时优雅关闭所有组件。
func runTUI(listenAddr, dataDir string) error {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("无法创建数据目录: %w", err)
	}

	dbPath := dataDir + "/panel.db"
	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("无法打开数据库: %w", err)
	}
	defer st.Close()

	apiToken, err := ensureToken(st, "api_token")
	if err != nil {
		return fmt.Errorf("无法初始化 API token: %w", err)
	}

	subToken, err := ensureToken(st, "sub_token")
	if err != nil {
		return fmt.Errorf("无法初始化订阅 token: %w", err)
	}

	plat := platform.Detect()
	var svcMgr service.Manager
	if mgr := service.NewManager(plat.InitSystem); mgr != nil {
		svcMgr = mgr
	}

	eng := engine.New(st)
	if err := eng.Start(); err != nil {
		slog.Warn("引擎启动失败（可能没有配置）", "error", err)
	}

	collector := stats.NewCollector(eng.Tracker(), st)
	collectorCtx, collectorCancel := context.WithCancel(context.Background())
	go collector.Run(collectorCtx, 60*time.Second)

	srv := api.NewServer(eng, st, svcMgr, listenAddr, apiToken, subToken)
	if err := srv.Start(); err != nil {
		collectorCancel()
		return fmt.Errorf("无法启动 API 服务: %w", err)
	}

	slog.Info("sing-box 面板已启动（TUI 模式）",
		"listen", listenAddr,
		"data_dir", dataDir,
	)

	// 阻塞直到 TUI 退出
	if err := tui.Run(listenAddr, apiToken, subToken); err != nil {
		slog.Error("TUI 退出异常", "error", err)
	}

	// TUI 退出后优雅关闭
	collectorCancel()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx) //nolint:errcheck
	eng.Stop()                //nolint:errcheck
	return nil
}

// runServe 启动守护进程，等待系统信号退出（无 TUI）。
func runServe(listenAddr, dataDir string) error {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("无法创建数据目录: %w", err)
	}

	dbPath := dataDir + "/panel.db"
	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("无法打开数据库: %w", err)
	}
	defer st.Close()

	apiToken, err := ensureToken(st, "api_token")
	if err != nil {
		return fmt.Errorf("无法初始化 API token: %w", err)
	}

	subToken, err := ensureToken(st, "sub_token")
	if err != nil {
		return fmt.Errorf("无法初始化订阅 token: %w", err)
	}

	plat := platform.Detect()
	var svcMgr service.Manager
	if mgr := service.NewManager(plat.InitSystem); mgr != nil {
		svcMgr = mgr
	}

	eng := engine.New(st)
	if err := eng.Start(); err != nil {
		slog.Warn("引擎启动失败（可能没有配置）", "error", err)
	}

	collector := stats.NewCollector(eng.Tracker(), st)
	collectorCtx, collectorCancel := context.WithCancel(context.Background())
	go collector.Run(collectorCtx, 60*time.Second)

	srv := api.NewServer(eng, st, svcMgr, listenAddr, apiToken, subToken)
	if err := srv.Start(); err != nil {
		collectorCancel()
		return fmt.Errorf("无法启动 API 服务: %w", err)
	}

	slog.Info("sing-box 面板已启动",
		"listen", listenAddr,
		"data_dir", dataDir,
		"init_system", plat.InitSystem,
	)
	fmt.Printf("\nAPI Token: %s\n", apiToken)
	fmt.Printf("API 地址: http://%s\n", listenAddr)
	fmt.Printf("订阅地址: http://%s/sub/%s\n\n", listenAddr, subToken)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {
		sig := <-sigCh
		switch sig {
		case syscall.SIGHUP:
			slog.Info("收到 SIGHUP，重载引擎...")
			if err := eng.Reload(); err != nil {
				slog.Error("重载失败", "error", err)
			}
		case syscall.SIGINT, syscall.SIGTERM:
			slog.Info("正在关闭...")
			collectorCancel()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			srv.Shutdown(shutdownCtx) //nolint:errcheck
			eng.Stop()                //nolint:errcheck
			slog.Info("已关闭")
			return nil
		}
	}
}

func ensureToken(st *store.Store, key string) (string, error) {
	token, err := st.GetSetting(key)
	if err != nil {
		return "", err
	}
	if token == "" {
		token = generateToken()
		if err := st.SetSetting(key, token); err != nil {
			return "", err
		}
	}
	return token, nil
}

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b) //nolint:errcheck
	return hex.EncodeToString(b)
}
