// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2026 Matthew Penner

package nxretry_test

import (
	"context"
	"testing"
	"time"

	"github.com/matthewpi/nxretry"
)

func TestRealTimer(t *testing.T) {
	timer := nxretry.NewRealTimer()
	if timer == nil {
		t.Error("expected timer to not be nil")
		return
	}

	// C should return nil if the timer has not been started yet.
	if timer.C() != nil {
		t.Error("expected timer.C() to return nil when the timer has not started")
		return
	}

	// Ensure Stop returns true if the timer has not been started yet.
	if !timer.Stop() {
		t.Error("expected timer.Stop() to return true when the timer has not started")
		return
	}

	//
	// Start the timer.
	//

	// We use a small duration here to ensure this test doesn't take too long.
	timer.Start(10 * time.Millisecond)

	// C should not nil if the timer has not been started yet.
	if timer.C() == nil {
		t.Error("expected timer.C() to not return nil after the timer has started")
		return
	}

	// Wait for the timer to complete.
	<-timer.C()

	// C should still not be nil.
	if timer.C() == nil {
		t.Error("expected timer.C() to not return nil after the timer has completed")
		return
	}

	//
	// Ensure the timer can be re-used when calling Start again.
	//

	// Start the timer for the second time.
	start := time.Now()
	timer.Start(10 * time.Millisecond)

	// Ensure the timer waited the correct amount of time.
	duration := (<-timer.C()).Sub(start).Round(time.Millisecond)

	// While I haven't seen this test ever return anything other than
	// 10 milliseconds, add in a margin of error just to be safe. stdlib
	// tests should cover the underlying time being correct, we just want
	// to make sure our wrapper is wired up correctly.
	if duration < 9*time.Millisecond || duration > 11*time.Millisecond {
		t.Errorf("expected timer duration to be 10ms, got \"%s\"", duration)
		return
	}

	//
	// Ensure the timer gets stopped when Stop is called.
	//

	timer.Start(10 * time.Millisecond)

	// This uses a context to Stop the timer inside the select to prevent a
	// deadlock, this is usually how most people will Stop a timer. Stopping
	// a timer from outside the consumer is never recommended. Stopping a timer
	// should be used solely to prevent the timer from firing and to clean-up
	// the resources in-use by the timer in the event of a cancellation.
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	defer close(done)
	go func(ctx context.Context, c <-chan time.Time, done chan<- struct{}) {
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				// Ensure the channel gets drained.
				<-c
			}
		case <-c:
			t.Error("timer should not fire")
		}

		done <- struct{}{}
	}(ctx, timer.C(), done)

	cancel()
	<-done
}

// mockTimer for tests.
type mockTimer struct {
	started bool
	stopped bool
	c       chan time.Time
}

var _ nxretry.Timer = (*mockTimer)(nil)

func newMockTimer() nxretry.Timer {
	return &mockTimer{
		started: false,
		stopped: false,
	}
}

func (t *mockTimer) C() <-chan time.Time {
	if !t.started {
		return nil
	}
	return t.c
}

func (t *mockTimer) Start(time.Duration) {
	if !t.started {
		t.started = true
		t.c = make(chan time.Time)
	}

	go func() {
		t.c <- time.Now()
	}()
}

func (t *mockTimer) Stop() bool {
	if t.stopped {
		return true
	}
	t.stopped = true
	return false
}
