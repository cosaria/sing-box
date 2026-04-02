package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/233boy/sing-box/internal/api"
	"github.com/233boy/sing-box/internal/engine"
	"github.com/233boy/sing-box/internal/platform"
	"github.com/233boy/sing-box/internal/service"
	"github.com/233boy/sing-box/internal/stats"
	"github.com/233boy/sing-box/internal/store"
	"github.com/spf13/cobra"

	_ "github.com/233boy/sing-box/internal/protocol"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "sing-box",
		Short: "sing-box 管理面板",
	}
	rootCmd.AddCommand(serveCmd())
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

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
			srv.Shutdown(shutdownCtx)  //nolint:errcheck
			eng.Stop()                 //nolint:errcheck
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
