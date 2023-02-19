// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package pmetric

import (
	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
)

// Gauge represents the type of a numeric metric that always exports the "current value" for every data point.
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewGauge function to create new instances.
// Important: zero-initialized instance is not valid for use.
type Gauge struct {
	orig *otlpmetrics.Gauge
}

func newGauge(orig *otlpmetrics.Gauge) Gauge {
	return Gauge{orig}
}

// NewGauge creates a new empty Gauge.
//
// This must be used only in testing code. Users should use "AppendEmpty" when part of a Slice,
// OR directly access the member if this is embedded in another struct.
func NewGauge() Gauge {
	return newGauge(&otlpmetrics.Gauge{})
}

// MoveTo moves all properties from the current struct overriding the destination and
// resetting the current instance to its zero value
func (ms Gauge) MoveTo(dest Gauge) {
	*dest.orig = *ms.orig
	*ms.orig = otlpmetrics.Gauge{}
}

// DataPoints returns the DataPoints associated with this Gauge.
func (ms Gauge) DataPoints() NumberDataPointSlice {
	return newNumberDataPointSlice(&ms.orig.DataPoints)
}

// CopyTo copies all properties from the current struct overriding the destination.
func (ms Gauge) CopyTo(dest Gauge) {
	copyOrigGauge(dest.orig, ms.orig)
}

func copyOrigGauge(dest, src *otlpmetrics.Gauge) {
	copyOrigNumberDataPointSlice(&dest.DataPoints, &src.DataPoints)
}
