package store

import (
	"testing"
)

func TestOpenInMemory(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) failed: %v", err)
	}
	defer s.Close()
}

func TestMigrationsAreIdempotent(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.migrate(); err != nil {
		t.Fatalf("second migrate() failed: %v", err)
	}
}

func TestSettings(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	val, err := s.GetSetting("nonexistent")
	if err != nil {
		t.Fatalf("GetSetting(nonexistent) error: %v", err)
	}
	if val != "" {
		t.Fatalf("expected empty string, got %q", val)
	}

	if err := s.SetSetting("api_token", "secret123"); err != nil {
		t.Fatalf("SetSetting error: %v", err)
	}
	val, err = s.GetSetting("api_token")
	if err != nil {
		t.Fatalf("GetSetting error: %v", err)
	}
	if val != "secret123" {
		t.Fatalf("expected 'secret123', got %q", val)
	}

	if err := s.SetSetting("api_token", "updated"); err != nil {
		t.Fatalf("SetSetting overwrite error: %v", err)
	}
	val, _ = s.GetSetting("api_token")
	if val != "updated" {
		t.Fatalf("expected 'updated', got %q", val)
	}
}
