package engine

import (
	"testing"

	_ "github.com/233boy/sing-box/internal/protocol"
	"github.com/233boy/sing-box/internal/store"
)

func TestBuildShadowsocksInbound(t *testing.T) {
	ib := &store.Inbound{
		ID: 1, Tag: "ss-12345", Protocol: "shadowsocks", Port: 12345,
		Settings: `{"method":"2022-blake3-aes-128-gcm","password":"550e8400-e29b-41d4-a716-446655440000"}`,
	}
	opt, err := buildInbound(ib)
	if err != nil {
		t.Fatalf("buildInbound error: %v", err)
	}
	if opt.Type != "shadowsocks" {
		t.Errorf("type = %q, want shadowsocks", opt.Type)
	}
	if opt.Tag != "ss-12345" {
		t.Errorf("tag = %q, want ss-12345", opt.Tag)
	}
}

func TestBuildVLESSInbound(t *testing.T) {
	ib := &store.Inbound{
		Tag: "vless-443", Protocol: "vless", Port: 443,
		Settings: `{"uuid":"550e8400-e29b-41d4-a716-446655440000","flow":"xtls-rprx-vision","reality":{"private_key":"testkey","public_key":"testpub","short_id":"abcd1234","handshake":{"server":"www.microsoft.com","server_port":443}}}`,
	}
	opt, err := buildInbound(ib)
	if err != nil {
		t.Fatalf("buildInbound error: %v", err)
	}
	if opt.Type != "vless" {
		t.Errorf("type = %q, want vless", opt.Type)
	}
}

func TestBuildTrojanInbound(t *testing.T) {
	ib := &store.Inbound{
		Tag: "trojan-443", Protocol: "trojan", Port: 443,
		Settings: `{"password":"550e8400-e29b-41d4-a716-446655440000"}`,
	}
	opt, err := buildInbound(ib)
	if err != nil {
		t.Fatalf("buildInbound error: %v", err)
	}
	if opt.Type != "trojan" {
		t.Errorf("type = %q, want trojan", opt.Type)
	}
}

func TestBuildUnsupportedProtocol(t *testing.T) {
	ib := &store.Inbound{Protocol: "unknown-proto", Settings: "{}"}
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
}

func TestBuildOptionsWithInbounds(t *testing.T) {
	inbounds := []*store.Inbound{
		{Tag: "ss-1000", Protocol: "shadowsocks", Port: 1000, Settings: `{"method":"2022-blake3-aes-128-gcm","password":"550e8400-e29b-41d4-a716-446655440000"}`},
		{Tag: "trojan-2000", Protocol: "trojan", Port: 2000, Settings: `{"password":"550e8400-e29b-41d4-a716-446655440000"}`},
	}
	opts, err := buildOptions(inbounds)
	if err != nil {
		t.Fatalf("buildOptions error: %v", err)
	}
	if len(opts.Inbounds) != 2 {
		t.Errorf("expected 2 inbounds, got %d", len(opts.Inbounds))
	}
}
