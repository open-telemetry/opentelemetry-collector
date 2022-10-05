// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pmetric // import "go.opentelemetry.io/collector/pdata/pmetric"

const noRecordValueMask = uint32(1)

var DefaultMetricDataPointFlags = MetricDataPointFlags(0)

// MetricDataPointFlags defines how a metric aggregator reports aggregated values.
// It describes how those values relate to the time interval over which they are aggregated.
type MetricDataPointFlags uint32

// NoRecordedValue returns true if the MetricDataPointFlags contains the NoRecordedValue flag.
func (ms MetricDataPointFlags) NoRecordedValue() bool {
	return uint32(ms)&noRecordValueMask != 0
}

// WithNoRecordedValue returns a new MetricDataPointFlags, with the NoRecordedValue flag set to the given value.
func (ms MetricDataPointFlags) WithNoRecordedValue(b bool) MetricDataPointFlags {
	orig := uint32(ms)
	if b {
		orig |= noRecordValueMask
	} else {
		orig &^= noRecordValueMask
	}
	return MetricDataPointFlags(orig)
}
