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
	_ MetricData = &gauge{}
	_ MetricData = &sum{}
	_ MetricData = &histogram{}
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
	case "gauge":
		md = &gauge{}
	case "sum":
		md = &sum{}
	case "histogram":
		md = &histogram{}
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

// Aggregated defines a metric aggregation type.
type Aggregated struct {
	// Aggregation describes if the aggregator reports delta changes
	// since last report time, or cumulative changes since a fixed start time.
	Aggregation string `yaml:"aggregation" validate:"oneof=delta cumulative"`
}

// Type gets the metric aggregation type.
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

// Mono defines the metric monotonicity.
type Mono struct {
	// Monotonic is true if the sum is monotonic.
	Monotonic bool `yaml:"monotonic"`
}

type gauge struct {
}

func (d gauge) Type() string {
	return "Gauge"
}

func (d gauge) HasMonotonic() bool {
	return false
}

func (d gauge) HasAggregated() bool {
	return false
}

type sum struct {
	Aggregated `yaml:",inline"`
	Mono       `yaml:",inline"`
}

func (d sum) Type() string {
	return "Sum"
}

func (d sum) HasMonotonic() bool {
	return true
}

func (d sum) HasAggregated() bool {
	return true
}

type histogram struct {
	Aggregated `yaml:",inline"`
}

func (d histogram) Type() string {
	return "Histogram"
}

func (d histogram) HasMonotonic() bool {
	return false
}

func (d histogram) HasAggregated() bool {
	return true
}
