package protocol

import (
	"testing"

	"github.com/233boy/sing-box/internal/store"
	"github.com/sagernet/sing-box/option"
)

// mockProtocol is a minimal Protocol implementation for testing the registry.
type mockProtocol struct{}

func (m *mockProtocol) Name() string       { return "mock" }
func (m *mockProtocol) DisplayName() string { return "Mock" }
func (m *mockProtocol) DefaultSettings(port uint16) (string, error) {
	return `{"key":"value"}`, nil
}
func (m *mockProtocol) BuildInbound(ib *store.Inbound) (option.Inbound, error) {
	return option.Inbound{Type: "mock", Tag: ib.Tag}, nil
}
func (m *mockProtocol) GenerateURL(ib *store.Inbound, host string) string {
	return "mock://" + host
}

func TestRegisterAndGet(t *testing.T) {
	registry = map[string]Protocol{}
	Register(&mockProtocol{})
	p := Get("mock")
	if p == nil {
		t.Fatal("Get(mock) returned nil")
	}
	if p.Name() != "mock" {
		t.Errorf("Name() = %q, want %q", p.Name(), "mock")
	}
}

func TestGetUnknown(t *testing.T) {
	registry = map[string]Protocol{}
	p := Get("nonexistent")
	if p != nil {
		t.Fatal("Get(nonexistent) should return nil")
	}
}

func TestAll(t *testing.T) {
	registry = map[string]Protocol{}
	Register(&mockProtocol{})
	all := All()
	if len(all) != 1 {
		t.Fatalf("All() returned %d protocols, want 1", len(all))
	}
}

func TestGenerateUUID(t *testing.T) {
	id := GenerateUUID()
	if len(id) != 36 {
		t.Errorf("UUID length = %d, want 36", len(id))
	}
	if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		t.Errorf("invalid UUID format: %s", id)
	}
}
