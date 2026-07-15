package store

import (
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestAddAndListTargets(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.AddTarget(Target{Name: "DC01", Host: "dc01.corp", Port: 389, Type: "LDAP", Enabled: true}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := s.AddTarget(Target{Name: "Exchange", Host: "mail.corp", Port: 443, Type: "https"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	targets, err := s.ListTargets()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("got %d targets, want 2", len(targets))
	}
	if targets[0].Type != "ldap" {
		t.Errorf("type not normalised to lowercase: %q", targets[0].Type)
	}
	if targets[0].CreatedAt.IsZero() {
		t.Error("created_at not set")
	}
}

func TestDeleteTarget(t *testing.T) {
	s := newTestStore(t)
	added, err := s.AddTarget(Target{Name: "X", Host: "x", Port: 22, Type: "ssh"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteTarget(added.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := s.DeleteTarget(added.ID); err != ErrNotFound {
		t.Errorf("second delete err = %v, want ErrNotFound", err)
	}
}

func TestValidateTarget(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.AddTarget(Target{Name: "", Host: "h"}); err == nil {
		t.Error("expected error for empty name")
	}
	if _, err := s.AddTarget(Target{Name: "n", Host: ""}); err == nil {
		t.Error("expected error for empty host")
	}
	if _, err := s.AddTarget(Target{Name: "n", Host: "h", Port: 70000}); err == nil {
		t.Error("expected error for invalid port")
	}
}
