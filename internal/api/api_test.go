package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/233boy/sing-box/internal/store"
)

type mockEngine struct {
	running   bool
	reloaded  int
	startErr  error
	reloadErr error
}

func (m *mockEngine) Start() error  { m.running = true; return m.startErr }
func (m *mockEngine) Stop() error   { m.running = false; return nil }
func (m *mockEngine) Reload() error { m.reloaded++; return m.reloadErr }
func (m *mockEngine) Running() bool { return m.running }

func setupTestServer(t *testing.T) (*Server, *store.Store, *mockEngine) {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	eng := &mockEngine{}
	srv := NewServer(eng, st, nil, "127.0.0.1:0", "test-token")
	return srv, st, eng
}

func TestGetStatusUnauthorized(t *testing.T) {
	srv, _, _ := setupTestServer(t)
	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestGetStatusAuthorized(t *testing.T) {
	srv, _, eng := setupTestServer(t)
	eng.running = true
	req := httptest.NewRequest("GET", "/api/status", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["engine"] != "running" {
		t.Errorf("engine = %v, want 'running'", resp["engine"])
	}
}

func TestCreateInbound(t *testing.T) {
	srv, _, eng := setupTestServer(t)
	body := `{"protocol":"shadowsocks","port":12345,"settings":{"method":"aes-256-gcm","password":"test"}}`
	req := httptest.NewRequest("POST", "/api/inbounds", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["tag"] != "ss-12345" {
		t.Errorf("tag = %v, want 'ss-12345'", resp["tag"])
	}
	if eng.reloaded != 1 {
		t.Errorf("engine.Reload() called %d times, want 1", eng.reloaded)
	}
}

func TestListInbounds(t *testing.T) {
	srv, st, _ := setupTestServer(t)
	st.CreateInbound(&store.Inbound{Tag: "ss-1000", Protocol: "shadowsocks", Port: 1000, Settings: "{}"})
	st.CreateInbound(&store.Inbound{Tag: "ss-2000", Protocol: "shadowsocks", Port: 2000, Settings: "{}"})

	req := httptest.NewRequest("GET", "/api/inbounds", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp []map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 inbounds, got %d", len(resp))
	}
}

func TestGetInbound(t *testing.T) {
	srv, st, _ := setupTestServer(t)
	ib := &store.Inbound{Tag: "ss-5000", Protocol: "shadowsocks", Port: 5000, Settings: "{}"}
	st.CreateInbound(ib)

	req := httptest.NewRequest("GET", "/api/inbounds/1", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestGetInboundNotFound(t *testing.T) {
	srv, _, _ := setupTestServer(t)
	req := httptest.NewRequest("GET", "/api/inbounds/999", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteInbound(t *testing.T) {
	srv, st, eng := setupTestServer(t)
	ib := &store.Inbound{Tag: "ss-6000", Protocol: "shadowsocks", Port: 6000, Settings: "{}"}
	st.CreateInbound(ib)

	req := httptest.NewRequest("DELETE", "/api/inbounds/1", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if eng.reloaded != 1 {
		t.Errorf("engine.Reload() called %d times, want 1", eng.reloaded)
	}
}

func TestReloadEndpoint(t *testing.T) {
	srv, _, eng := setupTestServer(t)
	req := httptest.NewRequest("POST", "/api/reload", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if eng.reloaded != 1 {
		t.Errorf("engine.Reload() called %d times, want 1", eng.reloaded)
	}
}
