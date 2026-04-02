package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

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
			dataDir, _ := cmd.Flags().GetString("data-dir")
			return runTUI(dataDir)
		},
	}
	rootCmd.Flags().String("data-dir", plat.DataDir, "数据目录")

	rootCmd.AddCommand(serveCmd())
	rootCmd.AddCommand(updateCmd())
	rootCmd.AddCommand(serviceCmd())
	rootCmd.AddCommand(versionCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// serveCmd 启动守护进程（sing-box 引擎），不启动 TUI。
// 守护进程将 PID 写入 <dataDir>/sing-box.pid，并监听 SIGHUP 重载。
func serveCmd() *cobra.Command {
	var dataDir string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "启动守护进程（sing-box 引擎）",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(dataDir)
		},
	}
	plat := platform.Detect()
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

// runTUI 打开数据库并直接进入 TUI 交互界面（无 HTTP API）。
// 配置变更通过向守护进程发送 SIGHUP 信号触发重载。
func runTUI(dataDir string) error {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("无法创建数据目录: %w", err)
	}

	dbPath := filepath.Join(dataDir, "panel.db")
	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("无法打开数据库: %w", err)
	}
	defer st.Close()

	return tui.Run(st, dataDir)
}

// runServe 启动守护进程：写 PID 文件，启动引擎 + 流量收集，等待信号。
// SIGHUP → 重载引擎，SIGINT/SIGTERM → 优雅退出。
func runServe(dataDir string) error {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("无法创建数据目录: %w", err)
	}

	// 写 PID 文件并加排他锁，防止双实例。
	pidFile := filepath.Join(dataDir, "sing-box.pid")
	pidLock, err := acquirePIDLock(pidFile)
	if err != nil {
		return fmt.Errorf("无法获取 PID 锁: %w", err)
	}
	defer func() {
		pidLock.Close()
		os.Remove(pidFile)
	}()

	dbPath := filepath.Join(dataDir, "panel.db")
	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("无法打开数据库: %w", err)
	}
	defer st.Close()

	eng := engine.New(st)
	if err := eng.Start(); err != nil {
		slog.Warn("引擎启动失败（可能没有配置）", "error", err)
	}

	collector := stats.NewCollector(eng.Tracker(), st)
	collectorCtx, collectorCancel := context.WithCancel(context.Background())
	if eng.Running() {
		go collector.Run(collectorCtx, 60*time.Second)
	} else {
		slog.Info("引擎未运行，流量收集器已跳过")
	}

	slog.Info("sing-box 守护进程已启动",
		"pid", os.Getpid(),
		"data_dir", dataDir,
	)

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
			eng.Stop() //nolint:errcheck
			slog.Info("已关闭")
			return nil
		}
	}
}

// acquirePIDLock 创建/打开 PID 文件并加排他锁，写入当前 PID。
// 返回的文件句柄必须由调用方持有直到进程退出，以维持锁。
func acquirePIDLock(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("另一个 sing-box serve 实例正在运行")
	}
	f.Truncate(0)
	f.WriteString(strconv.Itoa(os.Getpid()) + "\n")
	f.Sync()
	return f, nil
}
