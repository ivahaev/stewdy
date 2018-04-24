package stewdy

import (
	"os"
	"testing"
)

func TestOn(t *testing.T) {
	if len(eventHandlers) != 0 {
		t.Fatal("not emtpty eventHandlersMap")
	}

	h1 := func(t Target) {}
	h2 := func(t Target) {}
	On(EventOriginate, h1)
	if l := len(eventHandlers[EventOriginate]); l != 1 {
		t.Fatalf("unexpected len(eventHandlers[EventOriginate]), got: %d, want 1", l)
	}

	On(EventOriginate, h2)
	if l := len(eventHandlers[EventOriginate]); l != 2 {
		t.Fatalf("unexpected len(eventHandlers[EventOriginate]), got: %d, want 2", l)
	}

	On(EventOriginate, h1)
	On(EventOriginate, h2)
	if l := len(eventHandlers[EventOriginate]); l != 2 {
		t.Fatalf("unexpected len(eventHandlers[EventOriginate]), got: %d, want 2", l)
	}
}

func init() {
	if db == nil {
		os.Remove("./stewdy_test.db")
		Init("./stewdy_test.db")
	}
}
