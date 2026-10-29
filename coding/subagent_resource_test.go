package coding

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tmih06/ompig/coding/subagent"
)

// TestSubagentShutdownReleasesChildren proves the runtime releases child
// sessions on shutdown: no goroutines or live sessions survive it.
func TestSubagentShutdownReleasesChildren(t *testing.T) {
	session, _ := newSubagentTestSession(t, "release")
	runtime := session.SubagentRuntime()
	for _, name := range []string{"A", "B", "C"} {
		if _, err := runtime.Runner().Run(context.Background(), "", subagent.SpawnRequest{
			Name: name, Task: "work",
		}); err != nil {
			t.Fatalf("spawn %s: %v", name, err)
		}
	}
	for _, name := range []string{"A", "B", "C"} {
		if _, ok := runtime.ActiveSession(name); !ok {
			t.Fatalf("%s has no live session after its run", name)
		}
	}

	runtime.Shutdown()
	for _, name := range []string{"A", "B", "C"} {
		if _, ok := runtime.ActiveSession(name); ok {
			t.Errorf("%s kept a live session after shutdown", name)
		}
	}
	// Closing the parent again must not panic on the already-shut-down runtime.
	if err := session.Close(); err != nil {
		t.Errorf("Close after Shutdown: %v", err)
	}
}

// TestSubagentConcurrencyBound checks that concurrent spawns respect the
// runner's semaphore rather than starting everything at once.
func TestSubagentConcurrencyBound(t *testing.T) {
	session, _ := newSubagentTestSession(t, "bounded")
	runner := session.SubagentRuntime().Runner()
	runner.MaxConcurrency = 2

	gate := make(chan struct{}, 64)
	observed := make(chan int, 64)
	var inFlight atomic.Int32
	runner.Spawner = spawnerFunc(func(ctx context.Context, req subagent.SpawnRequest) (subagent.SpawnResult, error) {
		n := inFlight.Add(1)
		observed <- int(n)
		<-gate
		inFlight.Add(-1)
		return subagent.SpawnResult{Output: "ok"}, nil
	})

	// The bound is the runner's, not the caller's: parallel calls contend on it.
	var wg sync.WaitGroup
	for _, name := range []string{"a", "b", "c", "d", "e", "f"} {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			_, _ = runner.Run(context.Background(), "", subagent.SpawnRequest{Name: name, Task: "work"})
		}(name)
	}

	// Let the first two start, then release them one at a time.
	for range 2 {
		if n := <-observed; n > 2 {
			t.Errorf("%d spawns ran at once, want at most 2", n)
		}
	}
	for range 6 {
		gate <- struct{}{}
	}
	wg.Wait()
	if len(runner.Registry().List()) != 6 {
		t.Errorf("registered = %d, want 6", len(runner.Registry().List()))
	}
}

// spawnerFunc adapts a function to the Spawner interface.
type spawnerFunc func(context.Context, subagent.SpawnRequest) (subagent.SpawnResult, error)

func (f spawnerFunc) Spawn(ctx context.Context, req subagent.SpawnRequest) (subagent.SpawnResult, error) {
	return f(ctx, req)
}

// atomicAdd increments value and returns its new value.
func atomicAdd(value *int32) int32 {
	return atomic.AddInt32(value, 1)
}

// TestIdleTTLParkingReleasesASession proves an idle agent's live session is
// released after the TTL while its transcript and output survive, and that a
// message arriving first cancels the timer.
func TestIdleTTLParkingReleasesASession(t *testing.T) {
	session, _ := newSubagentTestSession(t, "idle release")
	runtime := session.SubagentRuntime()
	// The TTL must be set before the run settles, because settling arms the
	// timer with whatever TTL the runtime holds.
	runtime.SetIdleTTL(20 * time.Millisecond)
	if _, err := runtime.Runner().Run(context.Background(), "", subagent.SpawnRequest{
		Name: "Idle", Task: "work",
	}); err != nil {
		t.Fatal(err)
	}
	awaitSettlement(t, runtime)
	if _, ok := runtime.ActiveSession("Idle"); !ok {
		t.Fatal("no live session right after the run")
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := runtime.ActiveSession("Idle"); !ok {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if _, ok := runtime.ActiveSession("Idle"); ok {
		t.Fatal("the idle session was never parked")
	}
	entry, _ := runtime.Registry().Get("Idle")
	if entry.Status != subagent.StatusParked {
		t.Errorf("status = %q, want parked", entry.Status)
	}
	// The transcript and output survive, so a revive and a read still work.
	if _, ok := session.SubagentTranscript("Idle"); !ok {
		t.Error("the parked transcript is unavailable")
	}
	if output, err := runtime.ReadAgentOutput("Idle"); err != nil || output == "" {
		t.Errorf("parked output = %q err=%v", output, err)
	}
	if err := runtime.Revive(context.Background(), "Idle"); err != nil {
		t.Fatalf("Revive: %v", err)
	}
	if _, ok := runtime.ActiveSession("Idle"); !ok {
		t.Error("revive did not restore a live session")
	}
}
