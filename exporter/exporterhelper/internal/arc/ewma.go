// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package arc // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/arc"

import "math" // robustEWMA tracks an exponentially-weighted moving *mean* and a robust
// dispersion proxy: the EWMA of absolute residuals, i.e., E[ |x - mean_prev| ].
// This behaves similarly to a mean + K*sigma threshold but is cheaper and avoids
// squaring; it is less sensitive to outliers than naive variance updates.

type robustEWMA struct {
	alpha  float64
	init   bool
	mean   float64
	absDev float64 // EWMA of absolute residuals relative to previous mean
}

func newRobustEWMA(alpha float64) robustEWMA {
	return robustEWMA{alpha: alpha}
}

func (e *robustEWMA) initialized() bool { return e.init }

// update returns the new mean and absDev after ingesting x.
func (e *robustEWMA) update(x float64) (float64, float64) {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return e.mean, e.absDev
	}

	if !e.init {
		e.mean = x
		e.absDev = 0
		e.init = true
		return e.mean, e.absDev
	}
	prev := e.mean
	e.mean = e.alpha*x + (1-e.alpha)*e.mean
	resid := x - prev
	if resid < 0 {
		resid = -resid
	}
	e.absDev = e.alpha*resid + (1-e.alpha)*e.absDev
	return e.mean, e.absDev
}
