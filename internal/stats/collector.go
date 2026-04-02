package stats

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/cosaria/sing-box/internal/store"
)

type Collector struct {
	tracker  *Tracker
	store    *store.Store
	mu       sync.Mutex
	lastSnap map[string]TrafficCounter
}

func NewCollector(tracker *Tracker, st *store.Store) *Collector {
	return &Collector{
		tracker:  tracker,
		store:    st,
		lastSnap: make(map[string]TrafficCounter),
	}
}

func (c *Collector) Flush() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	current := c.tracker.Snapshot()
	for tag, cur := range current {
		last := c.lastSnap[tag]
		deltaUp := cur.Upload - last.Upload
		deltaDown := cur.Download - last.Download
		if deltaUp < 0 {
			deltaUp = cur.Upload
		}
		if deltaDown < 0 {
			deltaDown = cur.Download
		}
		if deltaUp == 0 && deltaDown == 0 {
			continue
		}
		if err := c.store.InsertTrafficLog(tag, deltaUp, deltaDown); err != nil {
			slog.Error("failed to insert traffic log", "tag", tag, "error", err)
			continue
		}
	}
	c.lastSnap = current
	return nil
}

func (c *Collector) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			c.Flush()
			return
		case <-ticker.C:
			c.Flush()
		}
	}
}
