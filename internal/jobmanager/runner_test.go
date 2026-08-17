package jobmanager

import (
	"testing"
	"time"
)

func TestRunner_StreamsLogLines(t *testing.T) {
	r := NewRunner("/bin/sh", []string{"-c", "echo one; echo two"}, t.TempDir())
	var events []RawEvent
	err := r.Run(func(e RawEvent) { events = append(events, e) })
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(events) != 2 || events[0].Text != "one" || events[1].Text != "two" {
		t.Fatalf("unexpected events: %+v", events)
	}
	for _, e := range events {
		if e.Kind != "log" {
			t.Fatalf("expected kind=log for plain echo output, got %+v", e)
		}
	}
}

func TestRunner_Cancel_TerminatesProcess(t *testing.T) {
	r := NewRunner("/bin/sh", []string{"-c", "sleep 30"}, t.TempDir())
	runDone := make(chan error, 1)
	go func() {
		runDone <- r.Run(func(RawEvent) {})
	}()

	// give the process a moment to actually start before cancelling
	time.Sleep(200 * time.Millisecond)
	r.Cancel()

	select {
	case <-runDone:
		// terminated well before the 30s sleep would finish on its own
	case <-time.After(5 * time.Second):
		t.Fatal("process was not terminated by Cancel within 5s")
	}
}
