package store

import (
	"testing"
)

func TestInsertTrafficLog(t *testing.T) {
	s := mustOpenTestStore(t)
	err := s.InsertTrafficLog("ss-8388", 1024, 2048)
	if err != nil {
		t.Fatalf("InsertTrafficLog error: %v", err)
	}
}

func TestGetTrafficSummary(t *testing.T) {
	s := mustOpenTestStore(t)
	s.InsertTrafficLog("ss-8388", 1000, 2000)
	s.InsertTrafficLog("ss-8388", 500, 1000)
	s.InsertTrafficLog("vless-443", 300, 600)
	summaries, err := s.GetTrafficSummary()
	if err != nil {
		t.Fatalf("GetTrafficSummary error: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}
	var found bool
	for _, s := range summaries {
		if s.Tag == "ss-8388" {
			found = true
			if s.Upload != 1500 {
				t.Errorf("ss-8388 upload = %d, want 1500", s.Upload)
			}
			if s.Download != 3000 {
				t.Errorf("ss-8388 download = %d, want 3000", s.Download)
			}
		}
	}
	if !found {
		t.Error("ss-8388 not found in summaries")
	}
}

func TestGetTrafficByTag(t *testing.T) {
	s := mustOpenTestStore(t)
	s.InsertTrafficLog("ss-8388", 1000, 2000)
	s.InsertTrafficLog("ss-8388", 500, 1000)
	logs, err := s.GetTrafficByTag("ss-8388")
	if err != nil {
		t.Fatalf("GetTrafficByTag error: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logs))
	}
}

func TestGetTrafficByTagEmpty(t *testing.T) {
	s := mustOpenTestStore(t)
	logs, err := s.GetTrafficByTag("nonexistent")
	if err != nil {
		t.Fatalf("GetTrafficByTag error: %v", err)
	}
	if len(logs) != 0 {
		t.Fatalf("expected 0 logs, got %d", len(logs))
	}
}
