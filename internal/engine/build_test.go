package engine

import (
	"testing"

	"github.com/233boy/sing-box/internal/store"
)

func TestBuildShadowsocksInbound(t *testing.T) {
	ib := &store.Inbound{
		ID:       1,
		Tag:      "ss-12345",
		Protocol: "shadowsocks",
		Port:     12345,
		Settings: `{"method":"aes-256-gcm","password":"testpass"}`,
	}

	opt, err := buildInbound(ib)
	if err != nil {
		t.Fatalf("buildInbound error: %v", err)
	}
	if opt.Type != "shadowsocks" {
		t.Errorf("type = %q, want %q", opt.Type, "shadowsocks")
	}
	if opt.Tag != "ss-12345" {
		t.Errorf("tag = %q, want %q", opt.Tag, "ss-12345")
	}
}

func TestBuildShadowsocksInvalidJSON(t *testing.T) {
	ib := &store.Inbound{
		Protocol: "shadowsocks",
		Settings: `{invalid}`,
	}

	_, err := buildInbound(ib)
	if err == nil {
		t.Fatal("expected error for invalid JSON settings")
	}
}

func TestBuildUnsupportedProtocol(t *testing.T) {
	ib := &store.Inbound{
		Protocol: "unknown-proto",
		Settings: "{}",
	}

	_, err := buildInbound(ib)
	if err == nil {
		t.Fatal("expected error for unsupported protocol")
	}
}

func TestBuildOptionsEmpty(t *testing.T) {
	opts, err := buildOptions(nil)
	if err != nil {
		t.Fatalf("buildOptions(nil) error: %v", err)
	}
	if len(opts.Inbounds) != 0 {
		t.Errorf("expected 0 inbounds, got %d", len(opts.Inbounds))
	}
	if len(opts.Outbounds) != 1 {
		t.Fatalf("expected 1 outbound (direct), got %d", len(opts.Outbounds))
	}
	if opts.Outbounds[0].Type != "direct" {
		t.Errorf("outbound type = %q, want 'direct'", opts.Outbounds[0].Type)
	}
}

func TestBuildOptionsWithInbounds(t *testing.T) {
	inbounds := []*store.Inbound{
		{Tag: "ss-1000", Protocol: "shadowsocks", Port: 1000, Settings: `{"method":"aes-256-gcm","password":"p1"}`},
		{Tag: "ss-2000", Protocol: "shadowsocks", Port: 2000, Settings: `{"method":"chacha20-ietf-poly1305","password":"p2"}`},
	}

	opts, err := buildOptions(inbounds)
	if err != nil {
		t.Fatalf("buildOptions error: %v", err)
	}
	if len(opts.Inbounds) != 2 {
		t.Errorf("expected 2 inbounds, got %d", len(opts.Inbounds))
	}
}
