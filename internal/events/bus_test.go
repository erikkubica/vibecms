package events

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPublish_FanOut(t *testing.T) {
	bus := New()
	var aHits atomic.Int32
	var bHits atomic.Int32

	bus.Subscribe("user.created", func(_ string, _ Payload) { aHits.Add(1) })
	bus.Subscribe("user.created", func(_ string, _ Payload) { bHits.Add(1) })

	bus.Publish("user.created", Payload{"id": 1})

	// Publish runs handlers in goroutines; wait briefly for them.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if aHits.Load() == 1 && bHits.Load() == 1 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("expected both handlers to fire; got a=%d b=%d", aHits.Load(), bHits.Load())
}

func TestPublishSync_BlocksUntilDone(t *testing.T) {
	bus := New()
	completed := make(chan struct{})

	bus.Subscribe("slow.op", func(_ string, _ Payload) {
		time.Sleep(20 * time.Millisecond)
		close(completed)
	})

	start := time.Now()
	bus.PublishSync("slow.op", nil)
	elapsed := time.Since(start)

	if elapsed < 20*time.Millisecond {
		t.Fatalf("PublishSync returned before handler completed (%v)", elapsed)
	}
	select {
	case <-completed:
	default:
		t.Fatal("handler did not complete before PublishSync returned")
	}
}

func TestPublishCollect_OrderedNonEmpty(t *testing.T) {
	bus := New()
	bus.SubscribeResult("render", func(_ string, _ Payload) string { return "first" })
	bus.SubscribeResult("render", func(_ string, _ Payload) string { return "" }) // empty result is dropped
	bus.SubscribeResult("render", func(_ string, _ Payload) string { return "third" })

	results := bus.PublishCollect("render", nil)
	if len(results) != 2 {
		t.Fatalf("expected 2 non-empty results, got %d (%v)", len(results), results)
	}
	if results[0] != "first" || results[1] != "third" {
		t.Fatalf("expected [first, third], got %v", results)
	}
}

func TestPanicInHandler_DoesNotKillBus(t *testing.T) {
	bus := New()
	var safeFired atomic.Int32

	bus.Subscribe("danger", func(_ string, _ Payload) { panic("boom") })
	bus.Subscribe("danger", func(_ string, _ Payload) { safeFired.Add(1) })

	bus.Publish("danger", nil)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if safeFired.Load() == 1 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	// And a subsequent publish should still work — bus must be alive.
	bus.Publish("danger", nil)
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if safeFired.Load() == 2 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("safe handler did not fire twice after panic; got %d", safeFired.Load())
}

func TestUnsubscribe_DropsHandler(t *testing.T) {
	bus := New()
	var hits atomic.Int32
	unsub := bus.Subscribe("ping", func(_ string, _ Payload) { hits.Add(1) })

	bus.PublishSync("ping", nil)
	if got := hits.Load(); got != 1 {
		t.Fatalf("expected 1 hit before unsubscribe, got %d", got)
	}
	unsub()
	bus.PublishSync("ping", nil)
	if got := hits.Load(); got != 1 {
		t.Fatalf("expected handler to be dropped, got %d hits", got)
	}
	// Calling unsub twice must be safe.
	unsub()
}

func TestSubscribeAll_Unsubscribe(t *testing.T) {
	bus := New()
	var seen atomic.Int32
	unsub := bus.SubscribeAll(func(_ string, _ Payload) { seen.Add(1) })

	bus.PublishSync("a", nil)
	bus.PublishSync("b", nil)
	if seen.Load() != 2 {
		t.Fatalf("expected 2 hits across both events, got %d", seen.Load())
	}
	unsub()
	bus.PublishSync("c", nil)
	if seen.Load() != 2 {
		t.Fatalf("SubscribeAll handler still firing after unsubscribe (got %d)", seen.Load())
	}
}

func TestSubscribeResult_Unsubscribe(t *testing.T) {
	bus := New()
	unsub := bus.SubscribeResult("render", func(_ string, _ Payload) string { return "x" })

	if got := bus.PublishCollect("render", nil); len(got) != 1 {
		t.Fatalf("expected 1 result before unsubscribe, got %v", got)
	}
	unsub()
	if got := bus.PublishCollect("render", nil); len(got) != 0 {
		t.Fatalf("expected 0 results after unsubscribe, got %v", got)
	}
}

// TestConcurrent_SubscribeAndPublish runs Subscribes and Publishes from
// many goroutines under -race. Without proper locking inside the bus,
// the race detector flags the underlying slice mutations. Run with
// `go test -race ./internal/events/...` to exercise the assertion.
func TestConcurrent_SubscribeAndPublish(t *testing.T) {
	bus := New()
	const N = 100
	var wg sync.WaitGroup
	wg.Add(N * 2)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			bus.Subscribe("concurrent", func(_ string, _ Payload) {})
		}()
		go func() {
			defer wg.Done()
			bus.Publish("concurrent", nil)
		}()
	}
	wg.Wait()
	// We don't assert on hit counts — the test exists to prove the race
	// detector stays clean.
}

func TestHasHandlers(t *testing.T) {
	bus := New()
	if bus.HasHandlers("nope") {
		t.Fatal("expected no handlers for unknown action")
	}
	unsub := bus.Subscribe("known", func(_ string, _ Payload) {})
	if !bus.HasHandlers("known") {
		t.Fatal("expected handler after Subscribe")
	}
	unsub()
	if bus.HasHandlers("known") {
		t.Fatal("expected no handlers after Unsubscribe")
	}
}
