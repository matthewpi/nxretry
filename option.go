// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2026 Matthew Penner

package nxretry

import (
	"context"
)

// options for a [retry].
type options struct {
	// backoff is used to control the delay between attempts.
	backoff Backoff

	// maxAttempts is the maximum number of attempts that can occur. If set to 0
	// the maximum number of attempts will be unlimited.
	maxAttempts uint

	// contextFactory to provide a [context.Context] for each attempt.
	contextFactory ContextFactory

	// timer is used for mocking in unit tests. For normal use, this should
	// always be set to the result of [NewRealTimer].
	timer Timer
}

// newOptions creates a new [options] instance with any defaults.
func newOptions() *options {
	return &options{}
}

// setDefaults sets the defaults for [options]. This is expected to be used
// after applying any [Option] to ensure any required options are set.
func (o *options) setDefaults() {
	if o.timer == nil {
		o.timer = NewRealTimer()
	}

	if o.contextFactory == nil {
		o.contextFactory = func(ctx context.Context) (context.Context, context.CancelFunc) {
			return ctx, nil
		}
	}
}

// Option for a [Retry].
type Option interface {
	// apply applies the [Option] to the [options] for a [Retry].
	apply(*options)
}

// OptionFunc type is an adapter to allow the use of ordinary functions as an
// [Option]. If f is a function with the appropriate signature, `OptionFunc(f)`
// is an [Option] that calls f.
type OptionFunc func(o *options)

var _ Option = (*OptionFunc)(nil)

func (f OptionFunc) apply(o *options) { f(o) }

// MaxAttempts sets the maximum number of attempts that can occur. If set to 0
// the maximum number of attempts will be unlimited.
func MaxAttempts(maxAttempts uint) OptionFunc {
	return func(o *options) {
		o.maxAttempts = maxAttempts
	}
}

// ContextFactory is a function that takes a context and returns a modified
// context, optionally with an associated cancel function. It is required to
// return a non-nil [context.Context], but the [context.ContextFunc] may be
// `nil` or an empty function `func() { }`.
type ContextFactory func(context.Context) (context.Context, context.CancelFunc)

// WithContextFactory sets the [ContextFactory] used by a [Retry].
//
// The context factory is invoked before each attempt and the returned
// [context.Context] is passed through by the [Retry.Next] iterator. If
// a non-nil [context.CancelFunc] is provided by the context factory, it
// will be called after the current iteration completes.
func WithContextFactory(f ContextFactory) OptionFunc {
	return func(o *options) {
		o.contextFactory = f
	}
}

// WithTimer overrides the [Timer] used by a [Retry]. This should only ever
// be used for testing.
func WithTimer(t Timer) OptionFunc {
	return func(o *options) {
		o.timer = t
	}
}
