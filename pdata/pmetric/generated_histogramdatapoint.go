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
	"go.opentelemetry.io/collector/pdata/internal"
	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// HistogramDataPoint is a single data point in a timeseries that describes the time-varying values of a Histogram of values.
type HistogramDataPoint interface {
	commonHistogramDataPoint
	Attributes() pcommon.Map
	BucketCounts() pcommon.UInt64Slice
	ExplicitBounds() pcommon.Float64Slice
	Exemplars() ExemplarSlice
}

type MutableHistogramDataPoint interface {
	commonHistogramDataPoint
	MoveTo(dest MutableHistogramDataPoint)
	Attributes() pcommon.MutableMap
	SetStartTimestamp(pcommon.Timestamp)
	SetTimestamp(pcommon.Timestamp)
	SetCount(uint64)
	SetSum(float64)
	RemoveSum()
	BucketCounts() pcommon.MutableUInt64Slice
	ExplicitBounds() pcommon.MutableFloat64Slice
	Exemplars() MutableExemplarSlice
	SetFlags(DataPointFlags)
	SetMin(float64)
	RemoveMin()
	SetMax(float64)
	RemoveMax()
}

type commonHistogramDataPoint interface {
	getOrig() *otlpmetrics.HistogramDataPoint
	CopyTo(dest MutableHistogramDataPoint)
	StartTimestamp() pcommon.Timestamp
	Timestamp() pcommon.Timestamp
	Count() uint64
	Sum() float64
	HasSum() bool
	Flags() DataPointFlags
	Min() float64
	HasMin() bool
	Max() float64
	HasMax() bool
}

type immutableHistogramDataPoint struct {
	orig *otlpmetrics.HistogramDataPoint
}

type mutableHistogramDataPoint struct {
	immutableHistogramDataPoint
}

func newImmutableHistogramDataPoint(orig *otlpmetrics.HistogramDataPoint) immutableHistogramDataPoint {
	return immutableHistogramDataPoint{orig}
}

func newMutableHistogramDataPoint(orig *otlpmetrics.HistogramDataPoint) mutableHistogramDataPoint {
	return mutableHistogramDataPoint{immutableHistogramDataPoint{orig}}
}

func (ms immutableHistogramDataPoint) getOrig() *otlpmetrics.HistogramDataPoint {
	return ms.orig
}

// NewHistogramDataPoint creates a new empty HistogramDataPoint.
//
// This must be used only in testing code. Users should use "AppendEmpty" when part of a Slice,
// OR directly access the member if this is embedded in another struct.
func NewHistogramDataPoint() MutableHistogramDataPoint {
	return newMutableHistogramDataPoint(&otlpmetrics.HistogramDataPoint{})
}

// MoveTo moves all properties from the current struct overriding the destination and
// resetting the current instance to its zero value
func (ms mutableHistogramDataPoint) MoveTo(dest MutableHistogramDataPoint) {
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = otlpmetrics.HistogramDataPoint{}
}

// Attributes returns the Attributes associated with this HistogramDataPoint.
func (ms immutableHistogramDataPoint) Attributes() pcommon.Map {
	return internal.NewImmutableMap(&ms.getOrig().Attributes)
}

func (ms mutableHistogramDataPoint) Attributes() pcommon.MutableMap {
	return internal.NewMutableMap(&ms.getOrig().Attributes)
}

// StartTimestamp returns the starttimestamp associated with this HistogramDataPoint.
func (ms immutableHistogramDataPoint) StartTimestamp() pcommon.Timestamp {
	return pcommon.Timestamp(ms.orig.StartTimeUnixNano)
}

// SetStartTimestamp replaces the starttimestamp associated with this HistogramDataPoint.
func (ms mutableHistogramDataPoint) SetStartTimestamp(v pcommon.Timestamp) {
	ms.orig.StartTimeUnixNano = uint64(v)
}

// Timestamp returns the timestamp associated with this HistogramDataPoint.
func (ms immutableHistogramDataPoint) Timestamp() pcommon.Timestamp {
	return pcommon.Timestamp(ms.orig.TimeUnixNano)
}

// SetTimestamp replaces the timestamp associated with this HistogramDataPoint.
func (ms mutableHistogramDataPoint) SetTimestamp(v pcommon.Timestamp) {
	ms.orig.TimeUnixNano = uint64(v)
}

// Count returns the count associated with this HistogramDataPoint.
func (ms immutableHistogramDataPoint) Count() uint64 {
	return ms.getOrig().Count
}

// SetCount replaces the count associated with this HistogramDataPoint.
func (ms mutableHistogramDataPoint) SetCount(v uint64) {
	ms.getOrig().Count = v
}

// Sum returns the sum associated with this HistogramDataPoint.
func (ms immutableHistogramDataPoint) Sum() float64 {
	return ms.getOrig().GetSum()
}

// HasSum returns true if the HistogramDataPoint contains a
// Sum value, false otherwise.
func (ms immutableHistogramDataPoint) HasSum() bool {
	return ms.getOrig().Sum_ != nil
}

// SetSum replaces the sum associated with this HistogramDataPoint.
func (ms mutableHistogramDataPoint) SetSum(v float64) {
	ms.getOrig().Sum_ = &otlpmetrics.HistogramDataPoint_Sum{Sum: v}
}

// RemoveSum removes the sum associated with this HistogramDataPoint.
func (ms mutableHistogramDataPoint) RemoveSum() {
	ms.getOrig().Sum_ = nil
}

// BucketCounts returns the bucketcounts associated with this HistogramDataPoint.
func (ms immutableHistogramDataPoint) BucketCounts() pcommon.UInt64Slice {
	return internal.NewImmutableUInt64Slice(&ms.getOrig().BucketCounts)
}

func (ms mutableHistogramDataPoint) BucketCounts() pcommon.MutableUInt64Slice {
	return internal.NewMutableUInt64Slice(&ms.getOrig().BucketCounts)
}

// ExplicitBounds returns the explicitbounds associated with this HistogramDataPoint.
func (ms immutableHistogramDataPoint) ExplicitBounds() pcommon.Float64Slice {
	return internal.NewImmutableFloat64Slice(&ms.getOrig().ExplicitBounds)
}

func (ms mutableHistogramDataPoint) ExplicitBounds() pcommon.MutableFloat64Slice {
	return internal.NewMutableFloat64Slice(&ms.getOrig().ExplicitBounds)
}

// Exemplars returns the Exemplars associated with this HistogramDataPoint.
func (ms immutableHistogramDataPoint) Exemplars() ExemplarSlice {
	return newImmutableExemplarSlice(&ms.getOrig().Exemplars)
}

func (ms mutableHistogramDataPoint) Exemplars() MutableExemplarSlice {
	return newMutableExemplarSlice(&ms.getOrig().Exemplars)
}

// Flags returns the flags associated with this HistogramDataPoint.
func (ms immutableHistogramDataPoint) Flags() DataPointFlags {
	return DataPointFlags(ms.orig.Flags)
}

// SetFlags replaces the flags associated with this HistogramDataPoint.
func (ms mutableHistogramDataPoint) SetFlags(v DataPointFlags) {
	ms.orig.Flags = uint32(v)
}

// Min returns the min associated with this HistogramDataPoint.
func (ms immutableHistogramDataPoint) Min() float64 {
	return ms.getOrig().GetMin()
}

// HasMin returns true if the HistogramDataPoint contains a
// Min value, false otherwise.
func (ms immutableHistogramDataPoint) HasMin() bool {
	return ms.getOrig().Min_ != nil
}

// SetMin replaces the min associated with this HistogramDataPoint.
func (ms mutableHistogramDataPoint) SetMin(v float64) {
	ms.getOrig().Min_ = &otlpmetrics.HistogramDataPoint_Min{Min: v}
}

// RemoveMin removes the min associated with this HistogramDataPoint.
func (ms mutableHistogramDataPoint) RemoveMin() {
	ms.getOrig().Min_ = nil
}

// Max returns the max associated with this HistogramDataPoint.
func (ms immutableHistogramDataPoint) Max() float64 {
	return ms.getOrig().GetMax()
}

// HasMax returns true if the HistogramDataPoint contains a
// Max value, false otherwise.
func (ms immutableHistogramDataPoint) HasMax() bool {
	return ms.getOrig().Max_ != nil
}

// SetMax replaces the max associated with this HistogramDataPoint.
func (ms mutableHistogramDataPoint) SetMax(v float64) {
	ms.getOrig().Max_ = &otlpmetrics.HistogramDataPoint_Max{Max: v}
}

// RemoveMax removes the max associated with this HistogramDataPoint.
func (ms mutableHistogramDataPoint) RemoveMax() {
	ms.getOrig().Max_ = nil
}

// CopyTo copies all properties from the current struct overriding the destination.
func (ms immutableHistogramDataPoint) CopyTo(dest MutableHistogramDataPoint) {
	ms.Attributes().CopyTo(dest.Attributes())
	dest.SetStartTimestamp(ms.StartTimestamp())
	dest.SetTimestamp(ms.Timestamp())
	dest.SetCount(ms.Count())
	if ms.HasSum() {
		dest.SetSum(ms.Sum())
	}

	ms.BucketCounts().CopyTo(dest.BucketCounts())
	ms.ExplicitBounds().CopyTo(dest.ExplicitBounds())
	ms.Exemplars().CopyTo(dest.Exemplars())
	dest.SetFlags(ms.Flags())
	if ms.HasMin() {
		dest.SetMin(ms.Min())
	}

	if ms.HasMax() {
		dest.SetMax(ms.Max())
	}

}

func generateTestHistogramDataPoint() MutableHistogramDataPoint {
	tv := NewHistogramDataPoint()
	fillTestHistogramDataPoint(tv)
	return tv
}

func fillTestHistogramDataPoint(tv MutableHistogramDataPoint) {
	internal.FillTestMap(internal.NewMutableMap(&tv.getOrig().Attributes))
	tv.getOrig().StartTimeUnixNano = 1234567890
	tv.getOrig().TimeUnixNano = 1234567890
	tv.getOrig().Count = uint64(17)
	tv.orig.Sum_ = &otlpmetrics.HistogramDataPoint_Sum{Sum: float64(17.13)}
	tv.orig.BucketCounts = []uint64{1, 2, 3}
	tv.orig.ExplicitBounds = []float64{1, 2, 3}
	fillTestExemplarSlice(newMutableExemplarSlice(&tv.getOrig().Exemplars))
	tv.getOrig().Flags = 1
	tv.orig.Min_ = &otlpmetrics.HistogramDataPoint_Min{Min: float64(9.23)}
	tv.orig.Max_ = &otlpmetrics.HistogramDataPoint_Max{Max: float64(182.55)}
}
