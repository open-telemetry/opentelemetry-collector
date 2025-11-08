// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package arc // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/arc"

import (
	"math"
	"testing"
)

func TestMean(t *testing.T) {
	m := mean{}
	if got := m.average(); got != 0 {
		t.Fatalf("average of empty mean should be 0, got %f", got)
	}

	m.add(10)
	if got := m.average(); got != 10 {
		t.Fatalf("average after adding 10 should be 10, got %f", got)
	}

	m.add(20)
	if got := m.average(); got != 15 {
		t.Fatalf("average after adding 20 should be 15, got %f", got)
	}

	m.add(30)
	if got := m.average(); got != 20 {
		t.Fatalf("average after adding 30 should be 20, got %f", got)
	}
}

func TestEwmaVar(t *testing.T) {
	const alpha = 0.4
	const tolerance = 1e-9

	e := newEwmaVar(alpha)
	if e.initialized() {
		t.Fatal("newEwmaVar should not be initialized")
	}
	if e.alpha != alpha {
		t.Fatalf("alpha not set correctly, got %f, want %f", e.alpha, alpha)
	}

	// First update initializes
	e.update(10)
	if !e.initialized() {
		t.Fatal("ewmaVar should be initialized after first update")
	}
	if e.mean != 10 {
		t.Fatalf("initial mean should be first value, got %f, want 10", e.mean)
	}
	if e.variance != 0 {
		t.Fatalf("initial variance should be 0, got %f, want 0", e.variance)
	}

	// Second update
	// mean = 0.4 * 20 + 0.6 * 10 = 8 + 6 = 14
	// d = 20 - 10 = 10
	// variance = 0.4 * (10*10) + 0.6 * 0 = 40
	e.update(20)
	if math.Abs(e.mean-14.0) > tolerance {
		t.Fatalf("second mean incorrect, got %f, want 14.0", e.mean)
	}
	if math.Abs(e.variance-40.0) > tolerance {
		t.Fatalf("second variance incorrect, got %f, want 40.0", e.variance)
	}

	// Third update
	// mPrev = 14
	// mean = 0.4 * 15 + 0.6 * 14 = 6 + 8.4 = 14.4
	// d = 15 - 14 = 1
	// variance = 0.4 * (1*1) + 0.6 * 40 = 0.4 + 24 = 24.4
	e.update(15)
	if math.Abs(e.mean-14.4) > tolerance {
		t.Fatalf("third mean incorrect, got %f, want 14.4", e.mean)
	}
	if math.Abs(e.variance-24.4) > tolerance {
		t.Fatalf("third variance incorrect, got %f, want 24.4", e.variance)
	}
}
