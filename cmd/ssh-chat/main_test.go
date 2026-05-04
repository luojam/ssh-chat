package main

import "testing"

func TestNewSessionMemberIDCreatesOpaqueSessionID(t *testing.T) {
	first, err := newSessionMemberID()
	if err != nil {
		t.Fatalf("newSessionMemberID returned error: %v", err)
	}
	second, err := newSessionMemberID()
	if err != nil {
		t.Fatalf("newSessionMemberID returned error: %v", err)
	}

	if first == "" {
		t.Fatal("first ID should not be empty")
	}
	if second == "" {
		t.Fatal("second ID should not be empty")
	}
	if first == second {
		t.Fatalf("IDs should differ, got %q twice", first)
	}
}
