// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2026 Matthew Penner

package nxretry

import (
	"math"
	"time"
)

// Exponential is the implementation of an exponential [Backoff].
type Exponential struct {
	// Factor is the factor at which Min will increase after each failed attempt.
	Factor float64
	// Min is the initial backoff time to wait after the first failed attempt.
	Min time.Duration
	// Max is the maximum time to wait before retrying.
	Max time.Duration
}

var _ Backoff = Exponential{}

// Delay returns the [time.Duration] to wait before running the given attempt.
func (e Exponential) Delay(attempt uint) time.Duration {
	// maxInt64 is used to avoid overflowing a time.Duration (int64) value.
	const maxInt64 = float64(math.MaxInt64 - 512)

	// The first attempt should never have a delay.
	if attempt == 0 {
		return 0
	}

	factor := math.Pow(e.Factor, float64(attempt))
	durF := float64(e.Min) * factor
	if durF > maxInt64 {
		return e.Max
	}

	dur := time.Duration(durF)
	switch {
	case dur < e.Min:
		return e.Min
	case dur > e.Max:
		return e.Max
	default:
		return dur
	}
}

// apply implements the [Option] interface.
func (e Exponential) apply(o *options) {
	o.backoff = e
}
