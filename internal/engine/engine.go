package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/233boy/sing-box/internal/store"
	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
)

type Engine struct {
	store     *store.Store
	mu        sync.Mutex
	instance  *box.Box
	running   bool
	startedAt time.Time
}

func New(s *store.Store) *Engine {
	return &Engine{store: s}
}

func (e *Engine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("engine already running")
	}

	inbounds, err := e.store.ListInbounds()
	if err != nil {
		return fmt.Errorf("failed to list inbounds: %w", err)
	}

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

	e.instance = instance
	e.running = true
	e.startedAt = time.Now()
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

	inbounds, err := e.store.ListInbounds()
	if err != nil {
		return fmt.Errorf("failed to list inbounds: %w", err)
	}

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
		return fmt.Errorf("failed to create new sing-box instance: %w", err)
	}

	if err := instance.Start(); err != nil {
		instance.Close()
		return fmt.Errorf("failed to start new sing-box instance: %w", err)
	}

	e.instance = instance
	e.running = true
	e.startedAt = time.Now()
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
