package protocol

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/233boy/sing-box/internal/store"
)

func TestVLESSDefaultSettings(t *testing.T) {
	v := &VLESS{}
	settingsJSON, err := v.DefaultSettings(443)
	if err != nil {
		t.Fatalf("DefaultSettings error: %v", err)
	}
	var s vlessSettings
	if err := json.Unmarshal([]byte(settingsJSON), &s); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(s.UUID) != 36 {
		t.Errorf("UUID length = %d, want 36", len(s.UUID))
	}
	if s.Flow != "xtls-rprx-vision" {
		t.Errorf("flow = %q, want xtls-rprx-vision", s.Flow)
	}
	if s.Reality.PrivateKey == "" {
		t.Error("reality.private_key should not be empty")
	}
	if s.Reality.PublicKey == "" {
		t.Error("reality.public_key should not be empty")
	}
	if len(s.Reality.ShortID) != 8 {
		t.Errorf("reality.short_id length = %d, want 8", len(s.Reality.ShortID))
	}
	if s.Reality.Handshake.Server != "www.microsoft.com" {
		t.Errorf("handshake.server = %q, want www.microsoft.com", s.Reality.Handshake.Server)
	}
}

func TestVLESSBuildInbound(t *testing.T) {
	v := &VLESS{}
	settingsJSON, _ := v.DefaultSettings(443)
	ib := &store.Inbound{Tag: "vless-443", Protocol: "vless", Port: 443, Settings: settingsJSON}
	opt, err := v.BuildInbound(ib)
	if err != nil {
		t.Fatalf("BuildInbound error: %v", err)
	}
	if opt.Type != "vless" {
		t.Errorf("type = %q, want vless", opt.Type)
	}
	if opt.Tag != "vless-443" {
		t.Errorf("tag = %q, want vless-443", opt.Tag)
	}
}

func TestVLESSGenerateURL(t *testing.T) {
	v := &VLESS{}
	settingsJSON, _ := v.DefaultSettings(443)
	ib := &store.Inbound{Tag: "vless-443", Port: 443, Settings: settingsJSON}
	u := v.GenerateURL(ib, "1.2.3.4")
	if !strings.HasPrefix(u, "vless://") {
		t.Errorf("URL should start with vless://, got %q", u)
	}
	if !strings.Contains(u, "@1.2.3.4:443") {
		t.Errorf("URL should contain @1.2.3.4:443, got %q", u)
	}
	if !strings.Contains(u, "security=reality") {
		t.Errorf("URL should contain security=reality, got %q", u)
	}
	if !strings.Contains(u, "flow=xtls-rprx-vision") {
		t.Errorf("URL should contain flow=xtls-rprx-vision, got %q", u)
	}
	if !strings.Contains(u, "fp=chrome") {
		t.Errorf("URL should contain fp=chrome, got %q", u)
	}
	if !strings.Contains(u, "#vless-443") {
		t.Errorf("URL should contain #vless-443, got %q", u)
	}
}
