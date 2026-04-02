package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckLatestVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v0.2.0",
			"assets":   []map[string]string{},
		})
	}))
	defer ts.Close()
	v, url, err := checkLatestVersion(ts.URL)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if v != "v0.2.0" {
		t.Errorf("version = %q, want v0.2.0", v)
	}
	if url == "" {
		t.Error("download URL should not be empty")
	}
}

func TestCheckLatestVersionHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer ts.Close()
	_, _, err := checkLatestVersion(ts.URL)
	if err == nil {
		t.Fatal("expected error for non-200 status, got nil")
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		current, latest string
		want            bool
	}{
		{"v0.1.0", "v0.2.0", true},
		{"v0.2.0", "v0.2.0", false},
		{"v0.3.0", "v0.2.0", false},
		{"dev", "v0.1.0", true},
		{"v0.9.0", "v0.10.0", true},   // 确保不是字符串比较
		{"v1.0.0", "v0.99.0", false},  // 高主版本不降级
		{"v2.0.0", "v10.0.0", true},   // 双位数主版本
	}
	for _, tt := range tests {
		got := isNewer(tt.current, tt.latest)
		if got != tt.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
		}
	}
}
