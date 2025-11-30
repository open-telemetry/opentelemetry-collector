// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var reNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

type MetricName string

func (mn MetricName) Render() (string, error) {
	return FormatIdentifier(string(mn), true)
}

func (mn MetricName) RenderUnexported() (string, error) {
	return FormatIdentifier(string(mn), false)
}

type Metric struct {
	Signal `mapstructure:",squash"`

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

	// Override the default prefix for the metric name.
	Prefix string `mapstructure:"prefix"`
}

type Stability struct {
	Level component.StabilityLevel `mapstructure:"level"`
	From  string                   `mapstructure:"from"`
}

func (s Stability) String() string {
	if s.Level == component.StabilityLevelUndefined ||
		s.Level == component.StabilityLevelStable {
		return ""
	}
	if s.From != "" {
		return fmt.Sprintf(" [%s since %s]", s.Level.String(), s.From)
	}
	return fmt.Sprintf(" [%s]", s.Level.String())
}

func (s *Stability) Unmarshal(parser *confmap.Conf) error {
	if !parser.IsSet("level") {
		return errors.New("missing required field: `stability.level`")
	}
	return parser.Unmarshal(s)
}

func (m *Metric) validate(metricName MetricName, semConvVersion string) error {
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
	if m.SemanticConvention != nil {
		if err := validateSemConvMetricURL(m.SemanticConvention.SemanticConventionRef, semConvVersion, string(metricName)); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}

func metricAnchor(metricName string) string {
	m := strings.ToLower(strings.TrimSpace(metricName))
	m = reNonAlnum.ReplaceAllString(m, "")
	return "metric-" + m
}

// validateSemConvMetricURL verifies the URL matches exactly:
// https://github.com/open-telemetry/semantic-conventions/blob/<semConvVersion>/*#metric-<metricName>
func validateSemConvMetricURL(rawURL, semConvVersion, metricName string) error {
	if strings.TrimSpace(rawURL) == "" {
		return errors.New("url is empty")
	}
	if strings.TrimSpace(semConvVersion) == "" {
		return errors.New("semConvVersion is empty")
	}
	if strings.TrimSpace(metricName) == "" {
		return errors.New("metricName is empty")
	}
	semConvVersion = "v" + semConvVersion

	anchor := metricAnchor(metricName)
	// Build a strict regex that enforces https, repo, blob, given version, any doc path, and exact anchor.
	pattern := fmt.Sprintf(`^https://github\.com/open-telemetry/semantic-conventions/blob/%s/[^#\s]+#%s$`,
		semConvVersion,
		anchor,
	)
	re := regexp.MustCompile(pattern)
	if !re.MatchString(rawURL) {
		return fmt.Errorf(
			"invalid semantic-conventions URL: want https://github.com/open-telemetry/semantic-conventions/blob/%s/*#%s, got %q",
			semConvVersion, anchor, rawURL)
	}
	return nil
}

func (m *Metric) Unmarshal(parser *confmap.Conf) error {
	if !parser.IsSet("enabled") {
		return errors.New("missing required field: `enabled`")
	}
	if !parser.IsSet("stability") {
		return errors.New("missing required field: `stability`")
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

// String returns name of the datapoint type.
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

// String returns name of the datapoint type.
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
	instrumentName := cases.Title(language.English).String(d.BasicType())

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
	instrumentName := cases.Title(language.English).String(d.BasicType())

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
	return true
}

func (d *Histogram) Instrument() string {
	instrumentName := cases.Title(language.English).String(d.BasicType())
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
