package protocol

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/233boy/sing-box/internal/store"
)

func TestShadowsocksDefaultSettings(t *testing.T) {
	ss := &Shadowsocks{}
	settingsJSON, err := ss.DefaultSettings(8388)
	if err != nil {
		t.Fatalf("DefaultSettings error: %v", err)
	}
	var s shadowsocksSettings
	if err := json.Unmarshal([]byte(settingsJSON), &s); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if s.Method != "2022-blake3-aes-128-gcm" {
		t.Errorf("method = %q, want 2022-blake3-aes-128-gcm", s.Method)
	}
	if len(s.Password) != 36 {
		t.Errorf("password length = %d, want 36 (UUID)", len(s.Password))
	}
}

func TestShadowsocksBuildInbound(t *testing.T) {
	ss := &Shadowsocks{}
	ib := &store.Inbound{
		Tag: "ss-8388", Protocol: "shadowsocks", Port: 8388,
		Settings: `{"method":"2022-blake3-aes-128-gcm","password":"550e8400-e29b-41d4-a716-446655440000"}`,
	}
	opt, err := ss.BuildInbound(ib)
	if err != nil {
		t.Fatalf("BuildInbound error: %v", err)
	}
	if opt.Type != "shadowsocks" {
		t.Errorf("type = %q, want shadowsocks", opt.Type)
	}
	if opt.Tag != "ss-8388" {
		t.Errorf("tag = %q, want ss-8388", opt.Tag)
	}
}

func TestShadowsocksBuildInboundInvalidJSON(t *testing.T) {
	ss := &Shadowsocks{}
	ib := &store.Inbound{Settings: `{invalid}`}
	_, err := ss.BuildInbound(ib)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestShadowsocksGenerateURL(t *testing.T) {
	ss := &Shadowsocks{}
	ib := &store.Inbound{
		Tag: "ss-8388", Port: 8388,
		Settings: `{"method":"2022-blake3-aes-128-gcm","password":"550e8400-e29b-41d4-a716-446655440000"}`,
	}
	url := ss.GenerateURL(ib, "1.2.3.4")
	if !strings.HasPrefix(url, "ss://") {
		t.Errorf("URL should start with ss://, got %q", url)
	}
	if !strings.Contains(url, "@1.2.3.4:8388") {
		t.Errorf("URL should contain @1.2.3.4:8388, got %q", url)
	}
	if !strings.HasSuffix(url, "#ss-8388") {
		t.Errorf("URL should end with #ss-8388, got %q", url)
	}
	parts := strings.SplitN(strings.TrimPrefix(url, "ss://"), "@", 2)
	decoded, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("cannot decode base64 part: %v", err)
	}
	if !strings.Contains(string(decoded), "2022-blake3-aes-128-gcm") {
		t.Errorf("decoded part should contain method, got %q", string(decoded))
	}
}
