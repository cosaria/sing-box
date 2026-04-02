package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/233boy/sing-box/internal/protocol"
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
	srv := NewServer(eng, st, nil, "127.0.0.1:0", "test-token", "test-sub-token")
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
	if resp["tag"] != "shadowsocks-12345" {
		t.Errorf("tag = %v, want 'shadowsocks-12345'", resp["tag"])
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

func TestCreateInboundAutoSettings(t *testing.T) {
	srv, _, eng := setupTestServer(t)
	body := `{"protocol":"shadowsocks","port":8388}`
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
	if resp["tag"] != "shadowsocks-8388" {
		t.Errorf("tag = %v, want 'shadowsocks-8388'", resp["tag"])
	}
	settings, ok := resp["settings"].(string)
	if !ok || settings == "" || settings == "{}" {
		t.Errorf("settings should be auto-generated, got %v", resp["settings"])
	}
	if eng.reloaded != 1 {
		t.Errorf("engine.Reload() called %d times, want 1", eng.reloaded)
	}
}

func TestCreateInboundVLESS(t *testing.T) {
	srv, _, _ := setupTestServer(t)
	body := `{"protocol":"vless","port":443}`
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
	if resp["tag"] != "vless-443" {
		t.Errorf("tag = %v, want 'vless-443'", resp["tag"])
	}
}

func TestCreateInboundUnsupportedProtocol(t *testing.T) {
	srv, _, _ := setupTestServer(t)
	body := `{"protocol":"unknown","port":1234}`
	req := httptest.NewRequest("POST", "/api/inbounds", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGetStats(t *testing.T) {
	srv, st, _ := setupTestServer(t)
	st.InsertTrafficLog("ss-8388", 1000, 2000)
	st.InsertTrafficLog("ss-8388", 500, 1000)
	req := httptest.NewRequest("GET", "/api/stats", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp []map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(resp))
	}
}

func TestGetStatsByTag(t *testing.T) {
	srv, st, _ := setupTestServer(t)
	st.InsertTrafficLog("ss-8388", 1000, 2000)
	req := httptest.NewRequest("GET", "/api/stats/ss-8388", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSubscriptionValidToken(t *testing.T) {
	srv, st, _ := setupTestServer(t)
	st.CreateInbound(&store.Inbound{
		Tag: "shadowsocks-8388", Protocol: "shadowsocks", Port: 8388,
		Settings: `{"method":"2022-blake3-aes-128-gcm","password":"550e8400-e29b-41d4-a716-446655440000"}`,
	})
	req := httptest.NewRequest("GET", "/sub/test-sub-token", nil)
	req.Host = "1.2.3.4:9090"
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("content-type = %q, want text/plain", ct)
	}
	if w.Body.Len() == 0 {
		t.Error("response body should not be empty")
	}
}

func TestSubscriptionInvalidToken(t *testing.T) {
	srv, _, _ := setupTestServer(t)
	req := httptest.NewRequest("GET", "/sub/wrong-token", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestEngineStart(t *testing.T) {
	srv, _, eng := setupTestServer(t)
	req := httptest.NewRequest("POST", "/api/engine/start", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !eng.running {
		t.Error("engine should be running after start")
	}
}

func TestEngineStop(t *testing.T) {
	srv, _, eng := setupTestServer(t)
	eng.running = true
	req := httptest.NewRequest("POST", "/api/engine/stop", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if eng.running {
		t.Error("engine should be stopped after stop")
	}
}
