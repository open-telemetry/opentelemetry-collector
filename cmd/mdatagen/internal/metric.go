// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type MetricName string

func (mn MetricName) Render() (string, error) {
	return FormatIdentifier(string(mn), true)
}

func (mn MetricName) RenderUnexported() (string, error) {
	return FormatIdentifier(string(mn), false)
}

type Metric struct {
	// Enabled defines whether the metric is enabled by default.
	Enabled bool `mapstructure:"enabled"`

	// Warnings that will be shown to user under specified conditions.
	Warnings Warnings `mapstructure:"warnings"`

	// Description of the metric.
	Description string `mapstructure:"description"`

	// The stability level of the metric.
	Stability Stability `mapstructure:"stability"`

	// ExtendedDocumentation of the metric. If specified, this will
	// be appended to the description used in generated documentation.
	ExtendedDocumentation string `mapstructure:"extended_documentation"`

	// Optional can be used to specify metrics that may
	// or may not be present in all cases, depending on configuration.
	Optional bool `mapstructure:"optional"`

	// Unit of the metric.
	Unit *string `mapstructure:"unit"`

	// Sum stores metadata for sum metric type
	Sum *Sum `mapstructure:"sum,omitempty"`
	// Gauge stores metadata for gauge metric type
	Gauge *Gauge `mapstructure:"gauge,omitempty"`
	// Histogram stores metadata for histogram metric type
	Histogram *Histogram `mapstructure:"histogram,omitempty"`

	// Attributes is the list of attributes that the metric emits.
	Attributes []AttributeName `mapstructure:"attributes"`

	// Level specifies the minimum `configtelemetry.Level` for which
	// the metric will be emitted. This only applies to internal telemetry
	// configuration.
	Level configtelemetry.Level `mapstructure:"level"`
}

type Stability struct {
	Level string `mapstructure:"level"`
	From  string `mapstructure:"from"`
}

func (s Stability) String() string {
	if len(s.Level) == 0 || strings.EqualFold(s.Level, component.StabilityLevelStable.String()) {
		return ""
	}
	if len(s.From) > 0 {
		return fmt.Sprintf(" [%s since %s]", s.Level, s.From)
	}
	return fmt.Sprintf(" [%s]", s.Level)
}

func (m *Metric) validate() error {
	var errs error
	if m.Sum == nil && m.Gauge == nil && m.Histogram == nil {
		errs = errors.Join(errs, errors.New("missing metric type key, "+
			"one of the following has to be specified: sum, gauge, histogram"))
	}
	if (m.Sum != nil && m.Gauge != nil) || (m.Sum != nil && m.Histogram != nil) || (m.Gauge != nil && m.Histogram != nil) {
		errs = errors.Join(errs, errors.New("more than one metric type keys, "+
			"only one of the following has to be specified: sum, gauge, histogram"))
	}
	if m.Description == "" {
		errs = errors.Join(errs, errors.New(`missing metric description`))
	}
	if m.Unit == nil {
		errs = errors.Join(errs, errors.New(`missing metric unit`))
	}
	if m.Sum != nil {
		errs = errors.Join(errs, m.Sum.Validate())
	}
	if m.Gauge != nil {
		errs = errors.Join(errs, m.Gauge.Validate())
	}
	return errs
}

func (m *Metric) Unmarshal(parser *confmap.Conf) error {
	if !parser.IsSet("enabled") {
		return errors.New("missing required field: `enabled`")
	}
	return parser.Unmarshal(m)
}

func (m Metric) Data() MetricData {
	if m.Sum != nil {
		return m.Sum
	}
	if m.Gauge != nil {
		return m.Gauge
	}
	if m.Histogram != nil {
		return m.Histogram
	}
	return nil
}

// MetricData is generic interface for all metric datatypes.
type MetricData interface {
	Type() string
	HasMonotonic() bool
	HasAggregated() bool
	HasMetricInputType() bool
	Instrument() string
	IsAsync() bool
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

func (mit MetricInputType) Validate() error {
	if mit.InputType != "" && mit.InputType != "string" {
		return fmt.Errorf("invalid `input_type` value \"%v\", must be \"\" or \"string\"", mit.InputType)
	}
	return nil
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

var _ MetricData = (*Gauge)(nil)

type Gauge struct {
	MetricValueType `mapstructure:"value_type"`
	MetricInputType `mapstructure:",squash"`
	Async           bool `mapstructure:"async,omitempty"`
}

// Unmarshal is a custom unmarshaler for gauge. Needed mostly to avoid MetricValueType.Unmarshal inheritance.
func (d *Gauge) Unmarshal(parser *confmap.Conf) error {
	if err := d.MetricValueType.Unmarshal(parser); err != nil {
		return err
	}
	return parser.Unmarshal(d, confmap.WithIgnoreUnused())
}

func (d *Gauge) Type() string {
	return "Gauge"
}

func (d *Gauge) HasMonotonic() bool {
	return false
}

func (d *Gauge) HasAggregated() bool {
	return false
}

func (d *Gauge) Instrument() string {
	instrumentName := cases.Title(language.English).String(d.MetricValueType.BasicType())

	if d.Async {
		instrumentName += "Observable"
	}

	instrumentName += "Gauge"
	return instrumentName
}

func (d *Gauge) IsAsync() bool {
	return d.Async
}

var _ MetricData = (*Sum)(nil)

type Sum struct {
	AggregationTemporality `mapstructure:"aggregation_temporality"`
	Mono                   `mapstructure:",squash"`
	MetricValueType        `mapstructure:"value_type"`
	MetricInputType        `mapstructure:",squash"`
	Async                  bool `mapstructure:"async,omitempty"`
}

// Unmarshal is a custom unmarshaler for sum. Needed mostly to avoid MetricValueType.Unmarshal inheritance.
func (d *Sum) Unmarshal(parser *confmap.Conf) error {
	if err := d.MetricValueType.Unmarshal(parser); err != nil {
		return err
	}
	return parser.Unmarshal(d, confmap.WithIgnoreUnused())
}

// TODO: Currently, this func will not be called because of https://github.com/open-telemetry/opentelemetry-collector/issues/6671. Uncomment function and
// add a test case to Test_LoadMetadata for file no_monotonic.yaml once the issue is solved.
//
// Unmarshal is a custom unmarshaler for Mono.
// func (m *Mono) Unmarshal(parser *confmap.Conf) error {
// 	if !parser.IsSet("monotonic") {
// 		return errors.New("missing required field: `monotonic`")
// 	}
// 	return parser.Unmarshal(m)
// }

func (d *Sum) Type() string {
	return "Sum"
}

func (d *Sum) HasMonotonic() bool {
	return true
}

func (d *Sum) HasAggregated() bool {
	return true
}

func (d *Sum) Instrument() string {
	instrumentName := cases.Title(language.English).String(d.MetricValueType.BasicType())

	if d.Async {
		instrumentName += "Observable"
	}
	if !d.Monotonic {
		instrumentName += "UpDown"
	}
	instrumentName += "Counter"
	return instrumentName
}

func (d *Sum) IsAsync() bool {
	return d.Async
}

var _ MetricData = (*Histogram)(nil)

type Histogram struct {
	AggregationTemporality `mapstructure:"aggregation_temporality"`
	Mono                   `mapstructure:",squash"`
	MetricValueType        `mapstructure:"value_type"`
	MetricInputType        `mapstructure:",squash"`
	Async                  bool      `mapstructure:"async,omitempty"`
	Boundaries             []float64 `mapstructure:"bucket_boundaries"`
}

func (d *Histogram) Type() string {
	return "Histogram"
}

func (d *Histogram) HasMonotonic() bool {
	return false
}

func (d *Histogram) HasAggregated() bool {
	return false
}

func (d *Histogram) Instrument() string {
	instrumentName := cases.Title(language.English).String(d.MetricValueType.BasicType())
	return instrumentName + d.Type()
}

// Unmarshal is a custom unmarshaler for histogram. Needed mostly to avoid MetricValueType.Unmarshal inheritance.
func (d *Histogram) Unmarshal(parser *confmap.Conf) error {
	if err := d.MetricValueType.Unmarshal(parser); err != nil {
		return err
	}
	return parser.Unmarshal(d, confmap.WithIgnoreUnused())
}

func (d *Histogram) IsAsync() bool {
	return d.Async
}
