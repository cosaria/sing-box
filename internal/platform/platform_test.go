package platform

import (
	"runtime"
	"testing"
)

func TestDetectReturnsValidInfo(t *testing.T) {
	info := Detect()

	if info.OS == "" {
		t.Error("OS should not be empty")
	}
	if info.Arch == "" {
		t.Error("Arch should not be empty")
	}
	if info.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", info.OS, runtime.GOOS)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("Arch = %q, want %q", info.Arch, runtime.GOARCH)
	}
	if info.DataDir == "" {
		t.Error("DataDir should not be empty")
	}
	if info.BinPath == "" {
		t.Error("BinPath should not be empty")
	}
	if info.LogDir == "" {
		t.Error("LogDir should not be empty")
	}
}

func TestDetectInitSystemParsing(t *testing.T) {
	tests := []struct {
		name     string
		pid1     string
		expected string
	}{
		{"systemd", "systemd", "systemd"},
		{"openrc init", "init", "openrc"},
		{"openrc openrc-init", "openrc-init", "openrc"},
		{"unknown", "launchd", "unknown"},
		{"empty", "", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInitSystem(tt.pid1)
			if result != tt.expected {
				t.Errorf("parseInitSystem(%q) = %q, want %q", tt.pid1, result, tt.expected)
			}
		})
	}
}
