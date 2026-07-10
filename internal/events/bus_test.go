package events

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestEmitAndHandle(t *testing.T) {
	// Start event bus
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	Start(ctx)

	// Track received events
	var received []Event
	var mu sync.Mutex

	// Register handler
	On("test.event", func(e Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	})

	// Emit event
	Emit("test.event", map[string]string{"key": "value"})

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 1 {
		t.Errorf("Expected 1 event, got %d", len(received))
	}

	if received[0].Name != "test.event" {
		t.Errorf("Event name = %s, want test.event", received[0].Name)
	}
}

func TestMultipleHandlers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	Start(ctx)

	var count int
	var mu sync.Mutex

	// Register multiple handlers for same event
	On("multi.event", func(e Event) {
		mu.Lock()
		count++
		mu.Unlock()
	})
	On("multi.event", func(e Event) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	Emit("multi.event", nil)

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if count != 2 {
		t.Errorf("Expected 2 handler calls, got %d", count)
	}
}

func TestHandlerPanicRecovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	Start(ctx)

	var recovered bool
	var mu sync.Mutex

	// Handler that panics
	On("panic.event", func(e Event) {
		panic("test panic")
	})

	// Handler that runs after
	On("panic.event", func(e Event) {
		mu.Lock()
		recovered = true
		mu.Unlock()
	})

	// Should not crash
	Emit("panic.event", nil)

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if !recovered {
		t.Error("Second handler should have run despite first handler panic")
	}
}
