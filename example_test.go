package thunderstorm

import (
	"context"
	"strings"
	"testing"
)

func TestActionBasic(t *testing.T) {
	bus := New()
	ctx := context.Background()

	var executed []string

	bus.AddAction("test:event", 10, func(ctx context.Context, args ...interface{}) error {
		executed = append(executed, "handler1")
		return nil
	})

	bus.AddAction("test:event", 5, func(ctx context.Context, args ...interface{}) error {
		executed = append(executed, "handler2")
		return nil
	})

	err := bus.DoAction(ctx, "test:event", "data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Priority 5 should run before 10
	if len(executed) != 2 || executed[0] != "handler2" || executed[1] != "handler1" {
		t.Fatalf("wrong execution order: %v", executed)
	}
}

func TestFilterBasic(t *testing.T) {
	bus := New()
	ctx := context.Background()

	bus.AddFilter("title", 10, func(value interface{}, args ...interface{}) (interface{}, error) {
		return strings.ToUpper(value.(string)), nil
	})

	bus.AddFilter("title", 5, func(value interface{}, args ...interface{}) (interface{}, error) {
		return strings.TrimSpace(value.(string)), nil
	})

	result, err := bus.ApplyFilters(ctx, "title", "  hello world  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First: trim (priority 5) -> "hello world"
	// Second: uppercase (priority 10) -> "HELLO WORLD"
	if result != "HELLO WORLD" {
		t.Fatalf("wrong result: %v", result)
	}
}

func TestRemoveAction(t *testing.T) {
	bus := New()

	handler := func(ctx context.Context, args ...interface{}) error {
		return nil
	}

	bus.AddAction("test", 10, handler)
	if !bus.HasAction("test") {
		t.Fatal("action not added")
	}

	removed := bus.RemoveAction("test", handler)
	if !removed {
		t.Fatal("action not removed")
	}

	if bus.HasAction("test") {
		t.Fatal("action still exists after removal")
	}
}

func TestRemoveAllActions(t *testing.T) {
	bus := New()

	bus.AddAction("test", 10, func(_ context.Context, args ...interface{}) error { return nil })
	bus.AddAction("test", 20, func(_ context.Context, args ...interface{}) error { return nil })

	count := bus.RemoveAllActions("test")
	if count != 2 {
		t.Fatalf("expected 2 removed, got %d", count)
	}

	if bus.HasAction("test") {
		t.Fatal("actions still exist")
	}
}

func TestGetListeners(t *testing.T) {
	bus := New()
	ctx := context.Background()

	bus.AddAction("test", 10, func(_ context.Context, args ...interface{}) error { return nil })
	bus.AddAction("test", 5, func(_ context.Context, args ...interface{}) error { return nil })

	// Trigger sort by firing the action
	bus.DoAction(ctx, "test")

	listeners := bus.GetActionListeners("test")
	if len(listeners) != 2 {
		t.Fatalf("expected 2 listeners, got %d", len(listeners))
	}

	// Check priorities are sorted
	if listeners[0].Priority != 5 || listeners[1].Priority != 10 {
		t.Fatalf("listeners not sorted by priority: %v", listeners)
	}
}

func TestPanicRecovery(t *testing.T) {
	bus := New()
	ctx := context.Background()

	var executed []string

	bus.AddAction("test", 10, func(ctx context.Context, args ...interface{}) error {
		panic("oops")
	})

	bus.AddAction("test", 20, func(ctx context.Context, args ...interface{}) error {
		executed = append(executed, "safe")
		return nil
	})

	err := bus.DoAction(ctx, "test")
	if err == nil {
		t.Fatal("expected error from panic")
	}

	// Second handler should still execute
	if len(executed) != 1 || executed[0] != "safe" {
		t.Fatalf("safe handler didn't execute: %v", executed)
	}
}

