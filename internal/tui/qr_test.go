package tui_test

import (
	"strings"
	"testing"

	"github.com/233boy/sing-box/internal/tui"
)

func TestRenderQR(t *testing.T) {
	content := "ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ@127.0.0.1:1080#test"
	result := tui.RenderQR(content)
	if result == "" {
		t.Fatal("RenderQR() returned empty string for non-empty content")
	}
	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	if len(lines) < 5 {
		t.Errorf("RenderQR() returned %d lines, want at least 5", len(lines))
	}
	// 验证包含半块 Unicode 字符（QR 码特征）
	hasBlockChar := strings.ContainsAny(result, "█▀▄ ")
	if !hasBlockChar {
		t.Error("RenderQR() output does not contain expected block characters")
	}
}

func TestRenderQREmpty(t *testing.T) {
	result := tui.RenderQR("")
	if result != "" {
		t.Errorf("RenderQR(\"\") = %q, want empty string", result)
	}
}
