// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2026 Matthew Penner

package nxretry_test

import (
	"context"
	"time"

	"github.com/matthewpi/nxretry"
)

func ExampleNew() {
	r := nxretry.New(
		nxretry.MaxAttempts(3),
		nxretry.Exponential{
			Factor: 2,
			Min:    1 * time.Second,
			Max:    5 * time.Second,
		},
	)

	// Run code with the ability to retry, optionally using the provided context.
	for ctx := range r.Next(context.Background()) {
		_ = ctx

		// Do something.
		//
		// `break` on success (or if you don't want to retry anymore) and `continue` on failure.
	}
}

func ExampleNew_noContext() {
	r := nxretry.New(
		nxretry.MaxAttempts(3),
		nxretry.Exponential{
			Factor: 2,
			Min:    1 * time.Second,
			Max:    5 * time.Second,
		},
	)

	// Run code with the ability to retry.
	for range r.Next(context.Background()) {
		// Do something.
		//
		// `break` on success (or if you don't want to retry anymore) and `continue` on failure.
	}
}

func ExampleNew_contextFactory() {
	r := nxretry.New(
		nxretry.MaxAttempts(3),
		nxretry.Exponential{
			Factor: 2,
			Min:    1 * time.Second,
			Max:    5 * time.Second,
		},
		// For each attempt create a context with a dedicated timeout to prevent
		// any individual attempt from being attempted for too long.
		nxretry.WithContextFactory(func(ctx context.Context) (context.Context, context.CancelFunc) {
			return context.WithTimeout(ctx, 5*time.Second)
		}),
	)

	// Run code with the ability to retry, optionally using the provided context.
	for ctx := range r.Next(context.Background()) {
		_ = ctx // Context with timeout from the context factory.

		// Do something.
		//
		// `break` on success (or if you don't want to retry anymore) and `continue` on failure.
	}
}
