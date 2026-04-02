package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/233boy/sing-box/internal/stats"
	"github.com/233boy/sing-box/internal/store"
	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
)

const (
	crashLimit  = 3
	crashWindow = 60 * time.Second
)

type Engine struct {
	store       *store.Store
	mu          sync.Mutex
	instance    *box.Box
	running     bool
	startedAt   time.Time
	tracker     *stats.Tracker
	crashCount  int
	crashFirst  time.Time
	lastError   error
}

func New(s *store.Store) *Engine {
	return &Engine{store: s, tracker: stats.NewTracker()}
}

// resetCrashWindowIfExpired 在锁内调用，若距首次崩溃已超过 60s 则重置计数器。
func (e *Engine) resetCrashWindowIfExpired() {
	if e.crashCount > 0 && time.Since(e.crashFirst) > crashWindow {
		e.crashCount = 0
		e.crashFirst = time.Time{}
	}
}

// safeStart 包装 box.New + box.Start，捕获 panic 并更新崩溃计数。
// 调用方必须持有 e.mu。
func (e *Engine) safeStart(inbounds []*store.Inbound) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("engine panic: %v", r)
			e.crashCount++
			if e.crashCount == 1 {
				e.crashFirst = time.Now()
			}
			slog.Error("engine panic recovered", "panic", r, "crash_count", e.crashCount)
		}
	}()

	opts, err := buildOptions(inbounds)
	if err != nil {
		return fmt.Errorf("failed to build options: %w", err)
	}

	ctx := include.Context(context.Background())
	instance, err := box.New(box.Options{
		Context: ctx,
		Options: opts,
	})
	if err != nil {
		return fmt.Errorf("failed to create sing-box instance: %w", err)
	}

	if err := instance.Start(); err != nil {
		instance.Close()
		return fmt.Errorf("failed to start sing-box: %w", err)
	}

	instance.Router().AppendTracker(e.tracker)
	e.instance = instance
	e.running = true
	e.startedAt = time.Now()
	return nil
}

func (e *Engine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("engine already running")
	}

	e.resetCrashWindowIfExpired()
	if e.crashCount >= crashLimit {
		e.lastError = fmt.Errorf("engine crashed %d times in 60s, refusing to start", e.crashCount)
		return e.lastError
	}

	inbounds, err := e.store.ListInbounds()
	if err != nil {
		return fmt.Errorf("failed to list inbounds: %w", err)
	}

	if err := e.safeStart(inbounds); err != nil {
		e.lastError = err
		return err
	}

	e.lastError = nil
	slog.Info("engine started", "inbounds", len(inbounds))
	return nil
}

func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}

	if err := e.instance.Close(); err != nil {
		return fmt.Errorf("failed to close sing-box: %w", err)
	}

	e.instance = nil
	e.running = false
	slog.Info("engine stopped")
	return nil
}

func (e *Engine) Reload() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running && e.instance != nil {
		if err := e.instance.Close(); err != nil {
			slog.Warn("failed to close old instance during reload", "error", err)
		}
		e.instance = nil
		e.running = false
	}

	e.resetCrashWindowIfExpired()
	if e.crashCount >= crashLimit {
		e.lastError = fmt.Errorf("engine crashed %d times in 60s, refusing to reload", e.crashCount)
		return e.lastError
	}

	inbounds, err := e.store.ListInbounds()
	if err != nil {
		return fmt.Errorf("failed to list inbounds: %w", err)
	}

	if err := e.safeStart(inbounds); err != nil {
		e.lastError = err
		return err
	}

	e.lastError = nil
	slog.Info("engine reloaded", "inbounds", len(inbounds))
	return nil
}

func (e *Engine) Running() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

func (e *Engine) StartedAt() time.Time {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.startedAt
}

func (e *Engine) Tracker() *stats.Tracker {
	return e.tracker
}

// LastError 返回最近一次启动或重载的错误，供 API 查询。
func (e *Engine) LastError() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.lastError
}
