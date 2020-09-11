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

package main

import (
	"fmt"
)

var (
	_ MetricData = &intGauge{}
	_ MetricData = &intSum{}
	_ MetricData = &intHistogram{}
	_ MetricData = &doubleGauge{}
	_ MetricData = &doubleSum{}
	_ MetricData = &doubleHistogram{}
)

type ymlMetricData struct {
	MetricData `yaml:"-"`
}

// UnmarshalYAML converts the metrics.data map based on metrics.data.type.
func (e *ymlMetricData) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var m struct {
		Type string `yaml:"type"`
	}

	if err := unmarshal(&m); err != nil {
		return err
	}

	var md MetricData

	switch m.Type {
	case "int gauge":
		md = &intGauge{}
	case "int sum":
		md = &intSum{}
	case "int histogram":
		md = &intHistogram{}
	case "double gauge":
		md = &doubleGauge{}
	case "double sum":
		md = &doubleSum{}
	case "double histogram":
		md = &doubleHistogram{}
	default:
		return fmt.Errorf("metric data %q type invalid", m.Type)
	}

	if err := unmarshal(md); err != nil {
		return fmt.Errorf("unable to unmarshal data for type %q: %v", m.Type, err)
	}

	e.MetricData = md

	return nil
}

// MetricData is generic interface for all metric datatypes.
type MetricData interface {
	Type() string
	HasMonotonic() bool
	HasAggregated() bool
}

type Aggregated struct {
	// Aggregation describes if the aggregator reports delta changes
	// since last report time, or cumulative changes since a fixed start time.
	Aggregation string `yaml:"aggregation" validate:"oneof=delta cumulative"`
}

func (agg Aggregated) Type() string {
	switch agg.Aggregation {
	case "delta":
		return "pdata.AggregationTemporalityDelta"
	case "cumulative":
		return "pdata.AggregationTemporalityCumulative"
	default:
		return "pdata.AggregationTemporalityUnknown"
	}
}

type Mono struct {
	// Monotonic is true if the sum is monotonic.
	Monotonic bool `yaml:"monotonic"`
}

type intGauge struct {
}

func (i intGauge) Type() string {
	return "IntGauge"
}

func (i intGauge) HasMonotonic() bool {
	return false
}

func (i intGauge) HasAggregated() bool {
	return false
}

type doubleGauge struct {
}

func (d doubleGauge) Type() string {
	return "DoubleGauge"
}

func (d doubleGauge) HasMonotonic() bool {
	return false
}

func (d doubleGauge) HasAggregated() bool {
	return false
}

type intSum struct {
	Aggregated `yaml:",inline"`
	Mono       `yaml:",inline"`
}

func (i intSum) Type() string {
	return "IntSum"
}

func (i intSum) HasMonotonic() bool {
	return true
}

func (i intSum) HasAggregated() bool {
	return true
}

type doubleSum struct {
	Aggregated `yaml:",inline"`
	Mono       `yaml:",inline"`
}

func (d doubleSum) Type() string {
	return "DoubleSum"
}

func (d doubleSum) HasMonotonic() bool {
	return true
}

func (d doubleSum) HasAggregated() bool {
	return true
}

type intHistogram struct {
	Aggregated `yaml:",inline"`
}

func (i intHistogram) Type() string {
	return "IntHistogram"
}

func (i intHistogram) HasMonotonic() bool {
	return false
}

func (i intHistogram) HasAggregated() bool {
	return true
}

type doubleHistogram struct {
	Aggregated `yaml:",inline"`
}

func (d doubleHistogram) Type() string {
	return "DoubleHistogram"
}

func (d doubleHistogram) HasMonotonic() bool {
	return false
}

func (d doubleHistogram) HasAggregated() bool {
	return true
}
