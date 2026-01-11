// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2026 Matthew Penner

// Package nxretry implements a context-aware retrier with an optional and
// customizable backoff.
package nxretry

import (
	"context"
	"iter"
	"time"
)

// Backoff represents an abstract backoff implementation.
//
// This is used to abstract the duration to wait between attempts, to allow for
// different algorithms/implementations.
type Backoff interface {
	// TODO: this interface cannot be publicly implemented, making external
	// Backoff implementations impossible.
	Option

	// Delay returns the [time.Duration] to wait before running the given attempt.
	Delay(attempt uint) time.Duration
}

// Retry represents a retrier implementation.
type Retry interface {
	// Reset resets the [Retry] instance back to its original state.
	//
	// [Retry.Attempt], [Retry.Delay] and [Retry.Override], will be all be reset
	// to `0`.
	Reset()

	// Attempt returns the number of the current attempt.
	Attempt() uint

	// Delay returns the [time.Duration] to wait for the next attempt.
	//
	// The delay will be calculated using the [Backoff] configured on the [Retry]
	// instance. If [Retry.Override] has been used, its value will be returned
	// instead, until it has been consumed by an iteration of [Retry.Next].
	//
	// This function is useful for logging when the next attempt will occur.
	Delay() time.Duration

	// Next increments the attempt, then waits for the duration of the attempt.
	// Once the duration has passed, Next returns true. Next will return false if
	// the attempt will exceed the MaxAttempts limit or if the given context has
	// been canceled.
	//
	// This function was designed to be used as follows:
	//
	// 	r := New()
	// 	for ctx := range r.Next(ctx) {
	// 		// Do work, `continue` on soft-failure, `break` on success or non-retryable error.
	// 	}
	Next(ctx context.Context) iter.Seq[context.Context]

	// Override overrides the delay for the [Retry.Next] iteration.
	//
	// If a value of `0` is given, no override will be performed on the next
	// iteration.
	//
	// Calling [Retry.Override] multiple times before an iteration if safe, but
	// only the value from the last call will be used.
	//
	// The value of [Retry.Attempt] will still be incremented and count towards
	// the maximum attempt limit, however [Retry.Delay] will return a delay
	// as-if the overridden attempt never occurred. This is to allow for
	// temporary overrides without exceeding the maximum attempt limit or
	// changing the [Retry.Delay] for subsequent non-overridden attempts.
	//
	// [Retry.Delay] will temporarily return the value of `d`, until the
	// next iteration where the override will be reset. After the override
	// has been reset, [Retry.Delay] will return a delay as-if the overridden
	// attempt never occurred. This is to allow for temporary overrides without
	// exceeding the maximum attempt limit or affecting the delay for subsequent
	// non-overridden attempts.
	//
	// Subsequent iterations of [Retry.Next] will continue to use the [Backoff]
	// configured on the [Retry] instance, unless [Retry.Override] is called
	// again before the next iteration.
	Override(d time.Duration)
}

// retry represents a retrier as defined by the [Retry] interface.
type retry struct {
	*options

	// n is the current attempt and defaults to 0. This value is only used to
	// enforce the maximum attempts limit, while `delayIndex` is used to
	// calculate the delay for the attempt.
	//
	// This is necessary to support `delayIndex` as we don't want to affect
	// the delay given to us by our [Backoff] after an override, but we still
	// want to have overrides count towards the maximum attempts limit.
	n uint

	// delayIndex is the "attempt number" used to calculate the delay for
	// attempts.
	//
	// Previously `n` was used as both the attempt number and for calculating
	// the delay, however due to [Retry.Override] delayIndex was added.
	delayIndex uint

	// delayOverride is an override for the "Next" delay.
	//
	// If set to zero (the default), no override will be applied.
	delayOverride time.Duration
}

var _ Retry = (*retry)(nil)

// New creates a new retrier.
func New(opts ...Option) Retry {
	o := newOptions()
	for _, opt := range opts {
		opt.apply(o)
	}
	o.setDefaults()
	return &retry{options: o}
}

// Reset resets the [Retry] instance back to its original state.
//
// [Retry.Attempt], [Retry.Delay] and [Retry.Override], will be all be reset
// to `0`.
func (b *retry) Reset() {
	b.n = 0
	b.delayIndex = 0
	b.delayOverride = 0
}

// Attempt returns the number of the current attempt.
func (r *retry) Attempt() uint {
	return r.n
}

// Delay returns the [time.Duration] to wait for the next attempt.
//
// The delay will be calculated using the [Backoff] configured on the [Retry]
// instance. If [Retry.Override] has been used, its value will be returned
// instead, until it has been consumed by an iteration of [Retry.Next].
//
// This function is useful for logging when the next attempt will occur.
func (r *retry) Delay() time.Duration {
	if r.delayOverride > 0 {
		return r.delayOverride
	}
	return r.delay()
}

func (r *retry) delay() time.Duration {
	if r.backoff == nil {
		return 0
	}
	return r.backoff.Delay(r.delayIndex)
}

// Override overrides the delay for the [Retry.Next] iteration.
//
// If a value of `0` is given, no override will be performed on the next
// iteration.
//
// Calling [Retry.Override] multiple times before an iteration if safe, but
// only the value from the last call will be used.
//
// The value of [Retry.Attempt] will still be incremented and count towards
// the maximum attempt limit, however [Retry.Delay] will return a delay
// as-if the overridden attempt never occurred. This is to allow for
// temporary overrides without exceeding the maximum attempt limit or
// changing the [Retry.Delay] for subsequent non-overridden attempts.
//
// [Retry.Delay] will temporarily return the value of `d`, until the
// next iteration where the override will be reset. After the override
// has been reset, [Retry.Delay] will return a delay as-if the overridden
// attempt never occurred. This is to allow for temporary overrides without
// exceeding the maximum attempt limit or affecting the delay for subsequent
// non-overridden attempts.
//
// Subsequent iterations of [Retry.Next] will continue to use the [Backoff]
// configured on the [Retry] instance, unless [Retry.Override] is called
// again before the next iteration.
func (r *retry) Override(d time.Duration) {
	if d < 1 {
		d = 0
	}
	r.delayOverride = d
}

// Next increments the attempt, then waits for the duration of the attempt.
// Once the duration has passed, Next returns true. Next will return false if
// the attempt will exceed the MaxAttempts limit or if the given context has
// been canceled.
//
// This function was designed to be used as follows:
//
//	for ctx := range r.Next(ctx) {
//		// Do work, `continue` on soft-failure, `break` on success or non-retryable error.
//	}
func (r *retry) Next(ctx context.Context) iter.Seq[context.Context] {
	return func(yield func(context.Context) bool) {
		for {
			if !r.next(ctx) {
				return
			}

			// This should only occur during tests where we want to call Next
			// once and only once.
			if yield == nil {
				return
			}

			ctx, cancel := r.options.contextFactory(ctx)
			ok := yield(ctx)
			if cancel != nil {
				cancel()
			}
			if !ok {
				return
			}
		}
	}
}

func (r *retry) next(ctx context.Context) bool {
	// Check if we have exceeded the maximum attempts threshold.
	if r.maxAttempts != 0 && r.n >= r.maxAttempts {
		return false
	}

	// Get the delay for the attempt.
	var d time.Duration
	if r.delayOverride > 0 {
		// Override the delay.
		d = r.delayOverride

		// Clear the override so it doesn't apply to the next attempt.
		r.delayOverride = 0
	} else {
		// Get the delay for our current attempt.
		d = r.delay()

		// Increment the delay attempt, this MUST occur after we got the delay.
		r.delayIndex++
	}

	// Always increment the attempt, this is how we enforce the maximum attempt
	// limit.
	r.n++

	// If the delay is zero, bypass the timer, but still check the context.
	if d == 0 {
		select {
		case <-ctx.Done():
			return false
		default:
			return true
		}
	}

	r.timer.Start(d)
	select {
	case <-ctx.Done():
		// Stop the timer to release resources and prevent it from sending to a
		// channel we are not listening to anymore.
		if !r.timer.Stop() {
			// Drain the channel as per Go's documentation.
			<-r.timer.C()
		}
		return false
	case <-r.timer.C():
		return true
	}
}
