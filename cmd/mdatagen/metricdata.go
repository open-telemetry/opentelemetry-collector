// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var (
	_ MetricData = &gauge{}
	_ MetricData = &sum{}
)

// MetricData is generic interface for all metric datatypes.
type MetricData interface {
	Type() string
	HasMonotonic() bool
	HasAggregated() bool
	HasMetricInputType() bool
}

// AggregationTemporality defines a metric aggregation type.
type AggregationTemporality struct {
	// Aggregation describes if the aggregator reports delta changes
	// since last report time, or cumulative changes since a fixed start time.
	Aggregation pmetric.AggregationTemporality
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (agg *AggregationTemporality) UnmarshalText(text []byte) error {
	switch vtStr := string(text); vtStr {
	case "cumulative":
		agg.Aggregation = pmetric.AggregationTemporalityCumulative
	case "delta":
		agg.Aggregation = pmetric.AggregationTemporalityDelta
	default:
		return fmt.Errorf("invalid aggregation: %q", vtStr)
	}
	return nil
}

// String returns string representation of the aggregation temporality.
func (agg *AggregationTemporality) String() string {
	return agg.Aggregation.String()
}

// Mono defines the metric monotonicity.
type Mono struct {
	// Monotonic is true if the sum is monotonic.
	Monotonic bool `mapstructure:"monotonic"`
}

// MetricInputType defines the metric input value type
type MetricInputType struct {
	// InputType is the type the metric needs to be parsed from, options are "string"
	InputType string `mapstructure:"input_type"`
}

func (mit MetricInputType) HasMetricInputType() bool {
	return mit.InputType != ""
}

// Type returns name of the datapoint type.
func (mit MetricInputType) String() string {
	return mit.InputType
}

// MetricValueType defines the metric number type.
type MetricValueType struct {
	// ValueType is type of the metric number, options are "double", "int".
	ValueType pmetric.NumberDataPointValueType
}

func (mvt *MetricValueType) Unmarshal(parser *confmap.Conf) error {
	if !parser.IsSet("value_type") {
		return errors.New("missing required field: `value_type`")
	}
	return nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (mvt *MetricValueType) UnmarshalText(text []byte) error {
	switch vtStr := string(text); vtStr {
	case "int":
		mvt.ValueType = pmetric.NumberDataPointValueTypeInt
	case "double":
		mvt.ValueType = pmetric.NumberDataPointValueTypeDouble
	default:
		return fmt.Errorf("invalid value_type: %q", vtStr)
	}
	return nil
}

// Type returns name of the datapoint type.
func (mvt MetricValueType) String() string {
	return mvt.ValueType.String()
}

// BasicType returns name of a golang basic type for the datapoint type.
func (mvt MetricValueType) BasicType() string {
	switch mvt.ValueType {
	case pmetric.NumberDataPointValueTypeInt:
		return "int64"
	case pmetric.NumberDataPointValueTypeDouble:
		return "float64"
	case pmetric.NumberDataPointValueTypeEmpty:
		return ""
	default:
		return ""
	}
}

type gauge struct {
	MetricValueType `mapstructure:"value_type"`
	MetricInputType `mapstructure:",squash"`
}

// Unmarshal is a custom unmarshaler for gauge. Needed mostly to avoid MetricValueType.Unmarshal inheritance.
func (d *gauge) Unmarshal(parser *confmap.Conf) error {
	if err := d.MetricValueType.Unmarshal(parser); err != nil {
		return err
	}
	return parser.Unmarshal(d, confmap.WithErrorUnused())
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
	AggregationTemporality `mapstructure:"aggregation_temporality"`
	Mono                   `mapstructure:",squash"`
	MetricValueType        `mapstructure:"value_type"`
	MetricInputType        `mapstructure:",squash"`
}

// Unmarshal is a custom unmarshaler for sum. Needed mostly to avoid MetricValueType.Unmarshal inheritance.
func (d *sum) Unmarshal(parser *confmap.Conf) error {
	if !parser.IsSet("aggregation_temporality") {
		return errors.New("missing required field: `aggregation_temporality`")
	}
	if err := d.MetricValueType.Unmarshal(parser); err != nil {
		return err
	}
	return parser.Unmarshal(d, confmap.WithErrorUnused())
}

// TODO: Currently, this func will not be called because of https://github.com/open-telemetry/opentelemetry-collector/issues/6671. Uncomment function and
// add a test case to Test_loadMetadata for file no_monotonic.yaml once the issue is solved.
//
// Unmarshal is a custom unmarshaler for Mono.
// func (m *Mono) Unmarshal(parser *confmap.Conf) error {
// 	if !parser.IsSet("monotonic") {
// 		return errors.New("missing required field: `monotonic`")
// 	}
// 	return parser.Unmarshal(m, confmap.WithErrorUnused())
// }

func (d sum) Type() string {
	return "Sum"
}

func (d sum) HasMonotonic() bool {
	return true
}

func (d sum) HasAggregated() bool {
	return true
}
