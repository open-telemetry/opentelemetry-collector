// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package arc // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/arc"

type mean struct {
	n   int
	sum float64
}

func (m *mean) add(x float64) { m.n++; m.sum += x }
func (m *mean) average() float64 {
	if m.n == 0 {
		return 0
	}
	return m.sum / float64(m.n)
}

type ewmaVar struct {
	alpha    float64
	mean     float64
	variance float64
	init     bool
}

func newEwmaVar(alpha float64) ewmaVar { return ewmaVar{alpha: alpha} }
func (e *ewmaVar) initialized() bool   { return e.init }
func (e *ewmaVar) update(x float64) {
	if !e.init {
		e.mean, e.variance, e.init = x, 0, true
		return
	}
	mPrev := e.mean
	e.mean = e.alpha*x + (1-e.alpha)*e.mean
	d := x - mPrev
	e.variance = e.alpha*(d*d) + (1-e.alpha)*e.variance
}
