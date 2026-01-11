// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2026 Matthew Penner

package nxretry

import (
	"time"
)

// Timer is used as an abstraction to swap out the timer implementation used by
// Backoff. Most users will not need to implement this interface, it is used
// for mocking during tests.
type Timer interface {
	// C returns the channel where the current [time.Time] will be sent when the
	// timer fires. Calling [Timer.C] before [Timer.Start] will return nil, if
	// you ever find yourself nil-checking the result of this function, you are
	// likely using this interface incorrectly.
	C() <-chan time.Time

	// Start starts a timer using the specified duration. If Start is being
	// called for the first time it will create a new underlying timer,
	// otherwise it will reset the existing timer to a new duration.
	//
	// If you are re-using the same timer by calling Start multiple times,
	// ensure that the channel returned by [Timer.C] was drained or that
	// [Timer.Stop] was called properly according to its documentation.
	//
	// Essentially always ensure that the channel was drained if a value was
	// ever sent.
	Start(time.Duration)

	// Stop prevents the Timer from firing.
	//
	// It returns true if the call stops the timer, false if the timer has
	// already expired or been stopped.
	//
	// Stop does not close the channel, to prevent a read from the channel
	// succeeding incorrectly.
	//
	// To ensure the channel is empty after a call to Stop, check the
	// return value and drain the channel.
	// For example, assuming the program has not received from t.C already:
	//
	//	if !t.Stop() {
	//		<-t.C()
	//	}
	//
	// This cannot be done concurrent to other receives from the Timer's
	// channel or other calls to the Timer's Stop method.
	Stop() bool
}

// realTimer implements the Timer interface by wrapping a time#Timer.
type realTimer struct {
	timer *time.Timer
}

var _ Timer = (*realTimer)(nil)

// NewRealTimer returns a new real timer.
func NewRealTimer() Timer {
	return &realTimer{}
}

func (t *realTimer) C() <-chan time.Time {
	if t.timer == nil {
		return nil
	}
	return t.timer.C
}

func (t *realTimer) Start(d time.Duration) {
	if t.timer == nil {
		t.timer = time.NewTimer(d)
		return
	}
	t.timer.Reset(d)
}

func (t *realTimer) Stop() bool {
	if t.timer == nil {
		return true
	}
	return t.timer.Stop()
}
