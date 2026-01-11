// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2026 Matthew Penner

package nxretry_test

import (
	"context"
	"testing"
	"time"

	"github.com/matthewpi/nxretry"
)

func TestNew(t *testing.T) {
	const (
		_maxAttempts uint    = 3
		_factor      float64 = 2
		_min                 = 1 * time.Second
		_max                 = 5 * time.Second
	)

	r := nxretry.New(
		nxretry.MaxAttempts(_maxAttempts),
		nxretry.WithTimer(newMockTimer()),
		nxretry.Exponential{
			Factor: _factor,
			Min:    _min,
			Max:    _max,
		},
	)
	if r == nil {
		t.Error("expected retry to not be nil")
		return
	}

	for i, tc := range []struct {
		field  string
		expect any
		value  any
	}{
		// {
		// 	field:  "MaxAttempts",
		// 	expect: _maxAttempts,
		// 	value:  r.MaxAttempts,
		// },
		// {
		// 	field:  "Factor",
		// 	expect: _factor,
		// 	value:  r.Factor,
		// },
		// {
		// 	field:  "Min",
		// 	expect: _min,
		// 	value:  r.Min,
		// },
		// {
		// 	field:  "Max",
		// 	expect: _max,
		// 	value:  r.Max,
		// },
	} {
		if tc.expect != tc.value {
			t.Errorf("Test #%d: expected %s to be \"%s\", but got \"%s\"", i+1, tc.field, tc.expect, tc.value)
			continue
		}
	}
}

func TestRetry_Attempt(t *testing.T) {
	r := nxretry.New(
		nxretry.MaxAttempts(0),
		nxretry.WithTimer(newMockTimer()),
	)

	// Ensure attempt defaults to 0.
	if got := r.Attempt(); got != 0 {
		t.Errorf("Test #0: expected attempt to be \"%d\", but got \"%d\"", 0, got)
		return
	}

	// Run the first (0) attempt.
	r.Next(t.Context())(nil)

	// Ensure Next increments the attempt for the next run.
	if got := r.Attempt(); got != 1 {
		t.Errorf("Test #1: expected attempt to be \"%d\", but got \"%d\"", 1, got)
		return
	}
}

func TestRetry_Delay(t *testing.T) {
	e := nxretry.Exponential{
		Factor: 2,
		Min:    500 * time.Millisecond,
		Max:    3 * time.Second,
	}

	r := nxretry.New(nxretry.WithTimer(newMockTimer()), e)

	// Ensure first delay is 0.
	if delay := r.Delay(); delay != 0 {
		t.Errorf("Test #0: expected delay to be \"%s\", but got \"%s\"", time.Duration(0), delay)
		return
	}

	// Run the first attempt.
	r.Next(t.Context())(nil)

	// Ensure the delay matches what is expected from the underlying [Backoff] implementation.
	if expected, got := e.Delay(r.Attempt()), r.Delay(); got != expected {
		t.Errorf("Expected delay to be \"%s\", but got \"%s\"", expected, got)
	}
}

func TestRetry_Next(t *testing.T) {
	t.Run("Aborts before the first attempt when context is cancelled immediately", func(t *testing.T) {
		r := nxretry.New(nxretry.WithTimer(newMockTimer()))

		c := make(chan struct{})
		ctx, cancel := context.WithCancel(t.Context())
		go func(ctx context.Context) {
			for range r.Next(ctx) {
				t.Error("retry ran even though context was immediately cancelled")
			}
			close(c)
		}(ctx)

		cancel()
		<-c
	})

	t.Run("Aborts between attempts when context is cancelled", func(t *testing.T) {
		// This test sets time parameters to test the other branch of Next.
		// Next has two logic paths, one for when there is no duration and
		// another for when there is a duration.
		r := nxretry.New(
			nxretry.WithTimer(newMockTimer()),
			nxretry.MaxAttempts(0),
			nxretry.Exponential{
				Factor: 3,
				Min:    1 * time.Second,
				Max:    5 * time.Second,
			},
		)

		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan struct{})
		defer close(done)
		go func(ctx context.Context, done chan<- struct{}) {
			for range r.Next(ctx) {
				if r.Attempt() > 1 {
					t.Error("retry continued to run after context was cancelled")
					return
				}
				cancel()
			}

			done <- struct{}{}
		}(ctx, done)

		<-done
	})

	t.Run("Runs with MaxAttempts set to zero", func(t *testing.T) {
		r := nxretry.New(nxretry.WithTimer(newMockTimer()))

		i := 0
		r.Next(t.Context())(func(context.Context) bool {
			i++
			return false
		})
		switch {
		case i == 1:
			// test passed
		case i < 1:
			t.Error("Next doesn't run with MaxAttempts set to zero")
		case i > 1:
			t.Error("Next ran multiple times even though we returnd `false` to the iterator")
		}
	})

	t.Run("Aborts when MaxAttempt limit is reached", func(t *testing.T) {
		const maxAttempts = 5
		r := nxretry.New(nxretry.WithTimer(newMockTimer()), nxretry.MaxAttempts(maxAttempts))

		var i uint
		ctx := t.Context()
		for range r.Next(ctx) {
			i++
		}

		if i != maxAttempts {
			t.Errorf("expected number of attempts to be \"%d\", but got \"%d\"", maxAttempts, i)
		}
	})

	t.Run("Waits between attempts", func(t *testing.T) {
		r := nxretry.New(
			nxretry.WithTimer(newMockTimer()),
			nxretry.MaxAttempts(3),
			nxretry.Exponential{
				Factor: 2,
				Min:    5 * time.Millisecond,
				Max:    50 * time.Millisecond,
			},
		)

		var (
			i            uint
			lastDuration = r.Delay()
		)
		ctx := t.Context()
		for range r.Next(ctx) {
			d := r.Delay()
			if lastDuration >= d {
				t.Error("duration was expected to increase from the previous attempt")
				return
			}
			i++
			lastDuration = d
		}
	})
}

func TestRetry_Reset(t *testing.T) {
	r := nxretry.New(nxretry.WithTimer(newMockTimer()))

	// Run next to ensure the backoff is not in its default state.
	ctx := t.Context()
	r.Next(ctx)(nil)
	r.Next(ctx)(nil)

	if r.Attempt() == 0 {
		t.Error("retry attempt count is still at zero after being ran twice")
		return
	}

	// Reset the backoff.
	r.Reset()

	if r.Attempt() != 0 {
		t.Error("retry attempt count was not reset to zero")
		return
	}
}

func TestRetry_Override(t *testing.T) {
	r := nxretry.New(nxretry.WithTimer(newMockTimer()))

	// Override the delay.
	r.Override(5 * time.Second)
	if r.Delay() != 5*time.Second {
		t.Errorf("Override was not applied")
		return
	}

	// Clear the override.
	r.Override(0)
	if r.Delay() != 0 {
		t.Errorf("Override cannot be cleared")
		return
	}

	// Override the delay again.
	r.Override(5 * time.Second)

	ctx := t.Context()

	// Run the attempt using the overridden delay.
	r.Next(ctx)(nil)

	// Ensure the delay is no longer overridden.
	if r.Delay() != 0 {
		t.Errorf("Override was not cleared")
		return
	}
}
