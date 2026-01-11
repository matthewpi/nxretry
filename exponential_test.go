// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2026 Matthew Penner

package nxretry_test

import (
	"math"
	"testing"
	"time"

	"github.com/matthewpi/nxretry"
)

func TestExponential(t *testing.T) {
	t.Run("", func(t *testing.T) {
		e := nxretry.Exponential{
			Factor: 2,
			Min:    500 * time.Millisecond,
			Max:    3 * time.Second,
		}

		// Ensure the first delay is always zero.
		if got := e.Delay(0); got != 0 {
			t.Errorf("Expected the first delay to be \"%s\", but got \"%s\"", time.Duration(0), got)
		}

		// Ensure subsequent delays increase.
		for i := uint(1); i <= 3; i++ {
			if expected, got := time.Duration(e.Factor*float64(e.Min)*float64(i)), e.Delay(i); got != expected {
				t.Errorf("Expected the %d delay to be \"%s\", but got \"%s\"", i, expected, got)
			}
		}
	})

	t.Run("Enforces Minimum", func(t *testing.T) {
		e := nxretry.Exponential{
			Factor: 0.25,
			Min:    1 * time.Second,
			Max:    5 * time.Second,
		}
		if expected, got := e.Min, e.Delay(1); got != expected {
			t.Errorf("Expected delay to be \"%s\", but got \"%s\"", expected, got)
		}
	})

	t.Run("Enforces Maximum", func(t *testing.T) {
		e := nxretry.Exponential{
			Factor: 2,
			Min:    3 * time.Second,
			Max:    500 * time.Millisecond,
		}
		if expected, got := e.Max, e.Delay(1); got != expected {
			t.Errorf("Expected delay to be \"%s\", but got \"%s\"", expected, got)
		}
	})

	t.Run("Integer Overflow", func(t *testing.T) {
		e := nxretry.Exponential{
			Factor: math.MaxFloat64,
			Min:    3 * time.Second,
			Max:    500 * time.Millisecond,
		}
		if expected, got := e.Max, e.Delay(1); got != expected {
			t.Errorf("Expected delay to be \"%s\", but got \"%s\"", expected, got)
		}
	})
}
