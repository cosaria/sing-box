package stats

import (
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/sagernet/sing-box/adapter"
	N "github.com/sagernet/sing/common/network"
)

type TrafficCounter struct {
	Upload   int64
	Download int64
}

type inboundStats struct {
	Upload   atomic.Int64
	Download atomic.Int64
}

type Tracker struct {
	mu    sync.RWMutex
	stats map[string]*inboundStats
}

func NewTracker() *Tracker {
	return &Tracker{stats: make(map[string]*inboundStats)}
}

func (t *Tracker) getOrCreate(tag string) *inboundStats {
	t.mu.RLock()
	s, ok := t.stats[tag]
	t.mu.RUnlock()
	if ok {
		return s
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if s, ok := t.stats[tag]; ok {
		return s
	}
	s = &inboundStats{}
	t.stats[tag] = s
	return s
}

func (t *Tracker) Snapshot() map[string]TrafficCounter {
	t.mu.RLock()
	defer t.mu.RUnlock()
	snap := make(map[string]TrafficCounter, len(t.stats))
	for tag, s := range t.stats {
		snap[tag] = TrafficCounter{
			Upload:   s.Upload.Load(),
			Download: s.Download.Load(),
		}
	}
	return snap
}

func (t *Tracker) RoutedConnection(ctx context.Context, conn net.Conn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) net.Conn {
	tag := metadata.Inbound
	if tag == "" {
		return conn
	}
	s := t.getOrCreate(tag)
	return &countConn{Conn: conn, upload: &s.Upload, download: &s.Download}
}

func (t *Tracker) RoutedPacketConnection(ctx context.Context, conn N.PacketConn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) N.PacketConn {
	return conn
}

type countConn struct {
	net.Conn
	upload   *atomic.Int64
	download *atomic.Int64
}

func (c *countConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if n > 0 {
		c.download.Add(int64(n))
	}
	return
}

func (c *countConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	if n > 0 {
		c.upload.Add(int64(n))
	}
	return
}

func (c *countConn) WriteTo(w io.Writer) (n int64, err error) {
	if wt, ok := c.Conn.(io.WriterTo); ok {
		n, err = wt.WriteTo(w)
	} else {
		n, err = io.Copy(w, c.Conn)
	}
	if n > 0 {
		c.download.Add(n)
	}
	return
}
