package player

import "testing"

func TestNewNotifications(t *testing.T) {
	n := NewNotifications()
	if n == nil {
		t.Fatal("NewNotifications returned nil")
	}
}

func TestPushAndDrain(t *testing.T) {
	n := NewNotifications()
	n.Push("hello", "info")
	n.Push("error occurred", "error")

	toasts := n.Drain()
	if len(toasts) != 2 {
		t.Fatalf("got %d toasts, want 2", len(toasts))
	}
	if toasts[0].Message != "hello" {
		t.Errorf("toast[0].Message = %q", toasts[0].Message)
	}
	if toasts[0].Type != "info" {
		t.Errorf("toast[0].Type = %q", toasts[0].Type)
	}
	if toasts[1].Type != "error" {
		t.Errorf("toast[1].Type = %q", toasts[1].Type)
	}
}

func TestDrainEmpty(t *testing.T) {
	n := NewNotifications()
	toasts := n.Drain()
	if toasts != nil {
		t.Errorf("expected nil for empty drain, got %v", toasts)
	}
}

func TestDrainClearsQueue(t *testing.T) {
	n := NewNotifications()
	n.Push("msg", "info")

	_ = n.Drain()

	toasts := n.Drain()
	if toasts != nil {
		t.Errorf("expected nil after second drain, got %v", toasts)
	}
}

func TestPushAfterDrain(t *testing.T) {
	n := NewNotifications()
	n.Push("first", "info")
	_ = n.Drain()

	n.Push("second", "warn")
	toasts := n.Drain()
	if len(toasts) != 1 {
		t.Fatalf("got %d toasts, want 1", len(toasts))
	}
	if toasts[0].Message != "second" {
		t.Errorf("message = %q, want 'second'", toasts[0].Message)
	}
}
