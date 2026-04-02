package tui_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cosaria/sing-box/internal/tui"
)

// 构造测试服务器，返回固定 JSON 响应
func newTestServer(t *testing.T, mux *http.ServeMux) (*httptest.Server, *tui.Client) {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c := tui.NewClient(srv.URL, "testtoken")
	return srv, c
}

// 验证请求携带 Authorization header
func requireAuth(t *testing.T, r *http.Request) {
	t.Helper()
	if r.Header.Get("Authorization") != "Bearer testtoken" {
		t.Fatalf("missing or invalid Authorization header: %q", r.Header.Get("Authorization"))
	}
}

func TestStatus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		requireAuth(t, r)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"engine":   "running",
			"inbounds": 3,
		})
	})
	_, c := newTestServer(t, mux)

	st, err := c.Status()
	if err != nil {
		t.Fatalf("Status() error: %v", err)
	}
	if st.Engine != "running" {
		t.Errorf("Engine = %q, want %q", st.Engine, "running")
	}
	if st.Inbounds != 3 {
		t.Errorf("Inbounds = %d, want 3", st.Inbounds)
	}
}

func TestListInbounds(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/inbounds", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "not allowed", http.StatusMethodNotAllowed)
			return
		}
		requireAuth(t, r)
		inbounds := []tui.Inbound{
			{ID: 1, Tag: "ss-1080", Protocol: "shadowsocks", Port: 1080, Settings: "{}", CreatedAt: now.Format(time.RFC3339), UpdatedAt: now.Format(time.RFC3339)},
			{ID: 2, Tag: "ss-1081", Protocol: "shadowsocks", Port: 1081, Settings: "{}", CreatedAt: now.Format(time.RFC3339), UpdatedAt: now.Format(time.RFC3339)},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(inbounds) //nolint:errcheck
	})
	_, c := newTestServer(t, mux)

	list, err := c.ListInbounds()
	if err != nil {
		t.Fatalf("ListInbounds() error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d inbounds, want 2", len(list))
	}
	if list[0].Tag != "ss-1080" {
		t.Errorf("list[0].Tag = %q, want %q", list[0].Tag, "ss-1080")
	}
}

func TestCreateInbound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/inbounds", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "not allowed", http.StatusMethodNotAllowed)
			return
		}
		requireAuth(t, r)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body["protocol"] != "shadowsocks" {
			t.Errorf("protocol = %v, want shadowsocks", body["protocol"])
		}
		created := tui.Inbound{
			ID:       42,
			Tag:      "shadowsocks-9000",
			Protocol: "shadowsocks",
			Port:     9000,
			Settings: `{"method":"aes-256-gcm","password":"secret"}`,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(created) //nolint:errcheck
	})
	_, c := newTestServer(t, mux)

	ib, err := c.CreateInbound("shadowsocks", 9000)
	if err != nil {
		t.Fatalf("CreateInbound() error: %v", err)
	}
	if ib.ID != 42 {
		t.Errorf("ID = %d, want 42", ib.ID)
	}
	if ib.Port != 9000 {
		t.Errorf("Port = %d, want 9000", ib.Port)
	}
}

func TestDeleteInbound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/inbounds/7", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "not allowed", http.StatusMethodNotAllowed)
			return
		}
		requireAuth(t, r)
		w.WriteHeader(http.StatusNoContent)
	})
	_, c := newTestServer(t, mux)

	if err := c.DeleteInbound(7); err != nil {
		t.Fatalf("DeleteInbound() error: %v", err)
	}
}

func TestGetStats(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		requireAuth(t, r)
		stats := []tui.TrafficSummary{
			{Tag: "ss-1080", Upload: 1024, Download: 2048},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats) //nolint:errcheck
	})
	_, c := newTestServer(t, mux)

	stats, err := c.GetStats()
	if err != nil {
		t.Fatalf("GetStats() error: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("got %d stats, want 1", len(stats))
	}
	if stats[0].Upload != 1024 {
		t.Errorf("Upload = %d, want 1024", stats[0].Upload)
	}
}

func TestReload(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "not allowed", http.StatusMethodNotAllowed)
			return
		}
		requireAuth(t, r)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "reloaded"}) //nolint:errcheck
	})
	_, c := newTestServer(t, mux)

	if err := c.Reload(); err != nil {
		t.Fatalf("Reload() error: %v", err)
	}
}
