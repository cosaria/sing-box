package store

import (
	"testing"
)

func mustOpenTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) failed: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateAndGetInbound(t *testing.T) {
	s := mustOpenTestStore(t)

	ib := &Inbound{
		Tag:      "ss-12345",
		Protocol: "shadowsocks",
		Port:     12345,
		Settings: `{"method":"aes-256-gcm","password":"test123"}`,
	}
	if err := s.CreateInbound(ib); err != nil {
		t.Fatalf("CreateInbound error: %v", err)
	}
	if ib.ID == 0 {
		t.Fatal("expected ID to be set after create")
	}

	got, err := s.GetInbound(ib.ID)
	if err != nil {
		t.Fatalf("GetInbound error: %v", err)
	}
	if got.Tag != "ss-12345" {
		t.Errorf("tag = %q, want %q", got.Tag, "ss-12345")
	}
	if got.Protocol != "shadowsocks" {
		t.Errorf("protocol = %q, want %q", got.Protocol, "shadowsocks")
	}
	if got.Port != 12345 {
		t.Errorf("port = %d, want %d", got.Port, 12345)
	}
	if got.Settings != `{"method":"aes-256-gcm","password":"test123"}` {
		t.Errorf("settings = %q, want JSON", got.Settings)
	}
	if got.CreatedAt.IsZero() {
		t.Error("created_at should not be zero")
	}
}

func TestListInbounds(t *testing.T) {
	s := mustOpenTestStore(t)

	list, err := s.ListInbounds()
	if err != nil {
		t.Fatalf("ListInbounds error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 inbounds, got %d", len(list))
	}

	s.CreateInbound(&Inbound{Tag: "ss-1001", Protocol: "shadowsocks", Port: 1001, Settings: "{}"})
	s.CreateInbound(&Inbound{Tag: "ss-1002", Protocol: "shadowsocks", Port: 1002, Settings: "{}"})

	list, err = s.ListInbounds()
	if err != nil {
		t.Fatalf("ListInbounds error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 inbounds, got %d", len(list))
	}
}

func TestUpdateInbound(t *testing.T) {
	s := mustOpenTestStore(t)

	ib := &Inbound{Tag: "ss-2000", Protocol: "shadowsocks", Port: 2000, Settings: `{"method":"aes-256-gcm","password":"old"}`}
	s.CreateInbound(ib)

	ib.Port = 2001
	ib.Settings = `{"method":"aes-256-gcm","password":"new"}`
	if err := s.UpdateInbound(ib); err != nil {
		t.Fatalf("UpdateInbound error: %v", err)
	}

	got, _ := s.GetInbound(ib.ID)
	if got.Port != 2001 {
		t.Errorf("port = %d, want 2001", got.Port)
	}
	if got.Settings != `{"method":"aes-256-gcm","password":"new"}` {
		t.Errorf("settings not updated")
	}
}

func TestDeleteInbound(t *testing.T) {
	s := mustOpenTestStore(t)

	ib := &Inbound{Tag: "ss-3000", Protocol: "shadowsocks", Port: 3000, Settings: "{}"}
	s.CreateInbound(ib)

	if err := s.DeleteInbound(ib.ID); err != nil {
		t.Fatalf("DeleteInbound error: %v", err)
	}

	_, err := s.GetInbound(ib.ID)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got: %v", err)
	}
}

func TestCreateDuplicateTag(t *testing.T) {
	s := mustOpenTestStore(t)

	ib1 := &Inbound{Tag: "ss-4000", Protocol: "shadowsocks", Port: 4000, Settings: "{}"}
	s.CreateInbound(ib1)

	ib2 := &Inbound{Tag: "ss-4000", Protocol: "shadowsocks", Port: 4001, Settings: "{}"}
	err := s.CreateInbound(ib2)
	if err == nil {
		t.Fatal("expected error for duplicate tag")
	}
}
