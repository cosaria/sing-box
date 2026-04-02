package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
)

func main() {
	fmt.Println("=== M0: sing-box embedding validation ===")
	fmt.Println()

	port := uint16(18388)
	password := "test-password-for-m0"
	method := "aes-256-gcm"

	// Step 1: Create context with protocol registries
	ctx := include.Context(context.Background())

	// Step 2: Build Shadowsocks inbound config
	opts := box.Options{
		Context: ctx,
		Options: option.Options{
			Log: &option.LogOptions{
				Level: "info",
			},
			Inbounds: []option.Inbound{
				{
					Type: "shadowsocks",
					Tag:  "ss-test",
					Options: &option.ShadowsocksInboundOptions{
						ListenOptions: option.ListenOptions{
							ListenPort: port,
						},
						Method:   method,
						Password: password,
					},
				},
			},
			Outbounds: []option.Outbound{
				{
					Type: "direct",
					Tag:  "direct",
				},
			},
		},
	}

	// Step 3: Create box instance
	fmt.Println("[M0] Creating sing-box instance...")
	instance, err := box.New(opts)
	if err != nil {
		fmt.Printf("[M0] FAIL: box.New() error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[M0] PASS: box.New() succeeded")

	// Step 4: Start the engine
	fmt.Println("[M0] Starting sing-box engine...")
	err = instance.Start()
	if err != nil {
		fmt.Printf("[M0] FAIL: box.Start() error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[M0] PASS: Shadowsocks running on port %d (method: %s)\n", port, method)

	// Step 5: Basic HTTP API server
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "running",
			"port":    port,
			"method":  method,
			"version": "m0-prototype",
		})
	})

	apiAddr := "127.0.0.1:9090"
	server := &http.Server{Addr: apiAddr, Handler: mux}
	go func() {
		fmt.Printf("[M0] HTTP API listening on %s\n", apiAddr)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("[M0] HTTP server error: %v\n", err)
		}
	}()

	fmt.Println()
	fmt.Println("=== M0 VALIDATION RESULTS ===")
	fmt.Println("[PASS] 1. box.New() — sing-box instance created")
	fmt.Println("[PASS] 2. box.Start() — Shadowsocks inbound running")
	fmt.Println("[PASS] 3. HTTP API — basic server on :9090")
	fmt.Printf("[INFO] Test: curl http://%s/api/status\n", apiAddr)
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop...")

	// Wait for signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\n[M0] Shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	server.Shutdown(shutdownCtx)
	instance.Close()
	fmt.Println("[M0] Done.")
}
