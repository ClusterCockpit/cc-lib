package sinks

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
)

type fakeSink struct {
	sink
	block   chan struct{} // Write blocks until this channel is closed (nil = never block)
	mu      sync.Mutex
	written []lp.CCMessage
	closed  bool
}

func (s *fakeSink) Write(m lp.CCMessage) error {
	if s.block != nil {
		<-s.block
	}
	s.mu.Lock()
	s.written = append(s.written, m)
	s.mu.Unlock()
	return nil
}

func (s *fakeSink) Flush() error { return nil }

func (s *fakeSink) Close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
}

func (s *fakeSink) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.written)
}

func TestSinkManagerSlowSinkDoesNotStallFastSink(t *testing.T) {
	fast := &fakeSink{sink: sink{name: "fast"}}
	slow := &fakeSink{sink: sink{name: "slow"}, block: make(chan struct{})}
	AvailableSinks["testfast"] = func(name string, config json.RawMessage) (Sink, error) { return fast, nil }
	AvailableSinks["testslow"] = func(name string, config json.RawMessage) (Sink, error) { return slow, nil }
	defer delete(AvailableSinks, "testfast")
	defer delete(AvailableSinks, "testslow")

	config := json.RawMessage(`{
		"fastsink": {"type": "testfast"},
		"slowsink": {"type": "testslow", "queue_length": 1}
	}`)
	var wg sync.WaitGroup
	sm, err := New(&wg, config)
	if err != nil {
		t.Fatalf("failed to setup sink manager: %s", err.Error())
	}

	msgs, err := gen_messages(50)
	if err != nil {
		t.Fatalf("failed to generate messages: %s", err.Error())
	}
	input := make(chan lp.CCMessage, len(msgs))
	sm.AddInput(input)
	sm.Start()

	for _, m := range msgs {
		input <- m
	}

	// The fast sink must receive all messages although the slow sink is blocked
	deadline := time.Now().Add(5 * time.Second)
	for fast.count() < len(msgs) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := fast.count(); got != len(msgs) {
		t.Errorf("fast sink received %d of %d messages while slow sink was blocked", got, len(msgs))
	}

	// Unblock the slow sink so Close() can drain its queue
	close(slow.block)
	sm.Close()
	wg.Wait()

	if !fast.closed || !slow.closed {
		t.Error("sinks were not closed on shutdown")
	}
	if got := slow.count(); got == 0 {
		t.Error("slow sink received no messages at all")
	} else if got >= len(msgs) {
		t.Errorf("expected drops for the slow sink, but it received %d of %d messages", got, len(msgs))
	}
}
