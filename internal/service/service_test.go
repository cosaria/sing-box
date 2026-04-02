package service

import (
	"strings"
	"testing"
)

func TestSystemdUnit(t *testing.T) {
	content := systemdUnitContent("/usr/local/bin/sing-box", "/usr/local/etc/sing-box")
	if !strings.Contains(content, "ExecStart=/usr/local/bin/sing-box serve") {
		t.Error("unit should contain ExecStart with serve command")
	}
	if !strings.Contains(content, "--data-dir /usr/local/etc/sing-box") {
		t.Error("unit should contain --data-dir flag")
	}
	if !strings.Contains(content, "[Unit]") {
		t.Error("unit should have [Unit] section")
	}
}

func TestOpenRCInit(t *testing.T) {
	content := openrcInitContent("/usr/local/bin/sing-box", "/usr/local/etc/sing-box")
	if !strings.Contains(content, "command=\"/usr/local/bin/sing-box\"") {
		t.Error("init script should contain command path")
	}
	if !strings.Contains(content, "command_args=\"serve") {
		t.Error("init script should contain serve command")
	}
}

func TestNewManagerSystemd(t *testing.T) {
	m := NewManager("systemd")
	if m == nil {
		t.Fatal("NewManager(systemd) returned nil")
	}
}

func TestNewManagerOpenRC(t *testing.T) {
	m := NewManager("openrc")
	if m == nil {
		t.Fatal("NewManager(openrc) returned nil")
	}
}

func TestNewManagerUnknown(t *testing.T) {
	m := NewManager("unknown")
	if m != nil {
		t.Fatal("NewManager(unknown) should return nil")
	}
}
