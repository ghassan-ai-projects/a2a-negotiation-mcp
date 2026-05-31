package ipwhitelist

import (
	"testing"
)

func TestAddIP_Success(t *testing.T) {
	eng := NewEngine()
	err := eng.AddIP("192.168.1.1", "office-router")
	if err != nil {
		t.Fatalf("AddIP: %v", err)
	}

	entries := eng.List()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].IP != "192.168.1.1" {
		t.Errorf("expected IP 192.168.1.1, got %s", entries[0].IP)
	}
	if entries[0].Label != "office-router" {
		t.Errorf("expected label office-router, got %s", entries[0].Label)
	}
	if entries[0].CreatedAt == "" {
		t.Error("expected non-empty CreatedAt")
	}
}

func TestAddIP_Duplicate(t *testing.T) {
	eng := NewEngine()
	err := eng.AddIP("10.0.0.1", "vpn-gateway")
	if err != nil {
		t.Fatalf("first AddIP: %v", err)
	}

	err = eng.AddIP("10.0.0.1", "vpn-gateway")
	if err == nil {
		t.Fatal("expected error for duplicate IP")
	}
}

func TestRemoveIP_Success(t *testing.T) {
	eng := NewEngine()
	_ = eng.AddIP("10.0.0.1", "vpn-gateway")
	_ = eng.AddIP("192.168.1.1", "office-router")

	err := eng.RemoveIP("10.0.0.1")
	if err != nil {
		t.Fatalf("RemoveIP: %v", err)
	}

	entries := eng.List()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after removal, got %d", len(entries))
	}
	if entries[0].IP != "192.168.1.1" {
		t.Errorf("expected remaining IP 192.168.1.1, got %s", entries[0].IP)
	}
}

func TestRemoveIP_NotFound(t *testing.T) {
	eng := NewEngine()
	err := eng.RemoveIP("10.0.0.99")
	if err == nil {
		t.Fatal("expected error for non-existent IP")
	}
}

func TestList_MultipleEntries(t *testing.T) {
	eng := NewEngine()
	ips := []struct {
		ip    string
		label string
	}{
		{"10.0.0.1", "vpn-gateway"},
		{"192.168.1.1", "office-router"},
		{"172.16.0.1", "data-center"},
	}

	for _, e := range ips {
		if err := eng.AddIP(e.ip, e.label); err != nil {
			t.Fatalf("AddIP(%s): %v", e.ip, err)
		}
	}

	entries := eng.List()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	found := make(map[string]string)
	for _, e := range entries {
		found[e.IP] = e.Label
	}

	for _, e := range ips {
		if label, ok := found[e.ip]; !ok {
			t.Errorf("missing IP %s", e.ip)
		} else if label != e.label {
			t.Errorf("IP %s: expected label %s, got %s", e.ip, e.label, label)
		}
	}
}

func TestList_Empty(t *testing.T) {
	eng := NewEngine()
	entries := eng.List()
	if len(entries) != 0 {
		t.Errorf("expected empty list, got %d entries", len(entries))
	}
}
