package protocol

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/233boy/sing-box/internal/store"
)

func TestTrojanDefaultSettings(t *testing.T) {
	tr := &Trojan{}
	settingsJSON, err := tr.DefaultSettings(443)
	if err != nil {
		t.Fatalf("DefaultSettings error: %v", err)
	}
	var s trojanSettings
	if err := json.Unmarshal([]byte(settingsJSON), &s); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(s.Password) != 36 {
		t.Errorf("password length = %d, want 36 (UUID)", len(s.Password))
	}
}

func TestTrojanBuildInbound(t *testing.T) {
	tr := &Trojan{}
	ib := &store.Inbound{
		Tag: "trojan-443", Protocol: "trojan", Port: 443,
		Settings: `{"password":"550e8400-e29b-41d4-a716-446655440000"}`,
	}
	opt, err := tr.BuildInbound(ib)
	if err != nil {
		t.Fatalf("BuildInbound error: %v", err)
	}
	if opt.Type != "trojan" {
		t.Errorf("type = %q, want trojan", opt.Type)
	}
	if opt.Tag != "trojan-443" {
		t.Errorf("tag = %q, want trojan-443", opt.Tag)
	}
}

func TestTrojanBuildInboundInvalidJSON(t *testing.T) {
	tr := &Trojan{}
	ib := &store.Inbound{Settings: `{invalid}`}
	_, err := tr.BuildInbound(ib)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestTrojanGenerateURL(t *testing.T) {
	tr := &Trojan{}
	ib := &store.Inbound{
		Tag: "trojan-443", Port: 443,
		Settings: `{"password":"550e8400-e29b-41d4-a716-446655440000"}`,
	}
	u := tr.GenerateURL(ib, "1.2.3.4")
	if !strings.HasPrefix(u, "trojan://") {
		t.Errorf("URL should start with trojan://, got %q", u)
	}
	if !strings.Contains(u, "550e8400-e29b-41d4-a716-446655440000@1.2.3.4:443") {
		t.Errorf("URL should contain password@host:port, got %q", u)
	}
	if !strings.HasSuffix(u, "#trojan-443") {
		t.Errorf("URL should end with #trojan-443, got %q", u)
	}
}
