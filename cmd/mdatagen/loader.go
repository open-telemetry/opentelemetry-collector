// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

type metricName string

func (mn metricName) Render() (string, error) {
	return formatIdentifier(string(mn), true)
}

func (mn metricName) RenderUnexported() (string, error) {
	return formatIdentifier(string(mn), false)
}

type attributeName string

func (mn attributeName) Render() (string, error) {
	return formatIdentifier(string(mn), true)
}

func (mn attributeName) RenderUnexported() (string, error) {
	return formatIdentifier(string(mn), false)
}

// ValueType defines an attribute value type.
type ValueType struct {
	// ValueType is type of the attribute value.
	ValueType pcommon.ValueType
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (mvt *ValueType) UnmarshalText(text []byte) error {
	switch vtStr := string(text); vtStr {
	case "string":
		mvt.ValueType = pcommon.ValueTypeStr
	case "int":
		mvt.ValueType = pcommon.ValueTypeInt
	case "double":
		mvt.ValueType = pcommon.ValueTypeDouble
	case "bool":
		mvt.ValueType = pcommon.ValueTypeBool
	case "bytes":
		mvt.ValueType = pcommon.ValueTypeBytes
	case "slice":
		mvt.ValueType = pcommon.ValueTypeSlice
	case "map":
		mvt.ValueType = pcommon.ValueTypeMap
	default:
		return fmt.Errorf("invalid type: %q", vtStr)
	}
	return nil
}

// String returns capitalized name of the ValueType.
func (mvt ValueType) String() string {
	return strings.Title(strings.ToLower(mvt.ValueType.String())) // nolint SA1019
}

// Primitive returns name of primitive type for the ValueType.
func (mvt ValueType) Primitive() string {
	switch mvt.ValueType {
	case pcommon.ValueTypeStr:
		return "string"
	case pcommon.ValueTypeInt:
		return "int64"
	case pcommon.ValueTypeDouble:
		return "float64"
	case pcommon.ValueTypeBool:
		return "bool"
	case pcommon.ValueTypeBytes:
		return "[]byte"
	case pcommon.ValueTypeSlice:
		return "[]any"
	case pcommon.ValueTypeMap:
		return "map[string]any"
	case pcommon.ValueTypeEmpty:
		return ""
	default:
		return ""
	}
}

type metric struct {
	// Enabled defines whether the metric is enabled by default.
	Enabled bool `mapstructure:"enabled"`

	// Warnings that will be shown to user under specified conditions.
	Warnings warnings `mapstructure:"warnings"`

	// Description of the metric.
	Description string `mapstructure:"description"`

	// ExtendedDocumentation of the metric. If specified, this will
	// be appended to the description used in generated documentation.
	ExtendedDocumentation string `mapstructure:"extended_documentation"`

	// Unit of the metric.
	Unit *string `mapstructure:"unit"`

	// Sum stores metadata for sum metric type
	Sum *sum `mapstructure:"sum,omitempty"`
	// Gauge stores metadata for gauge metric type
	Gauge *gauge `mapstructure:"gauge,omitempty"`

	// Attributes is the list of attributes that the metric emits.
	Attributes []attributeName `mapstructure:"attributes"`
}

func (m *metric) Unmarshal(parser *confmap.Conf) error {
	if !parser.IsSet("enabled") {
		return errors.New("missing required field: `enabled`")
	}
	err := parser.Unmarshal(m, confmap.WithErrorUnused())
	if err != nil {
		return err
	}
	return nil
}
func (m metric) Data() MetricData {
	if m.Sum != nil {
		return m.Sum
	}
	if m.Gauge != nil {
		return m.Gauge
	}
	return nil
}

type warnings struct {
	// A warning that will be displayed if the field is enabled in user config.
	IfEnabled string `mapstructure:"if_enabled"`
	// A warning that will be displayed if `enabled` field is not set explicitly in user config.
	IfEnabledNotSet string `mapstructure:"if_enabled_not_set"`
	// A warning that will be displayed if the field is configured by user in any way.
	IfConfigured string `mapstructure:"if_configured"`
}

type attribute struct {
	// Description describes the purpose of the attribute.
	Description string `mapstructure:"description"`
	// NameOverride can be used to override the attribute name.
	NameOverride string `mapstructure:"name_override"`
	// Enabled defines whether the attribute is enabled by default.
	Enabled bool `mapstructure:"enabled"`
	// Enum can optionally describe the set of values to which the attribute can belong.
	Enum []string `mapstructure:"enum"`
	// Type is an attribute type.
	Type ValueType `mapstructure:"type"`
	// FullName is the attribute name populated from the map key.
	FullName attributeName `mapstructure:"-"`
	// Warnings that will be shown to user under specified conditions.
	Warnings warnings `mapstructure:"warnings"`
}

// Name returns actual name of the attribute that is set on the metric after applying NameOverride.
func (a attribute) Name() attributeName {
	if a.NameOverride != "" {
		return attributeName(a.NameOverride)
	}
	return a.FullName
}

func (a attribute) TestValue() string {
	if a.Enum != nil {
		return fmt.Sprintf(`"%s"`, a.Enum[0])
	}
	switch a.Type.ValueType {
	case pcommon.ValueTypeEmpty:
		return ""
	case pcommon.ValueTypeStr:
		return fmt.Sprintf(`"%s-val"`, a.FullName)
	case pcommon.ValueTypeInt:
		return fmt.Sprintf("%d", len(a.FullName))
	case pcommon.ValueTypeDouble:
		return fmt.Sprintf("%f", 0.1+float64(len(a.FullName)))
	case pcommon.ValueTypeBool:
		return fmt.Sprintf("%t", len(a.FullName)%2 == 0)
	case pcommon.ValueTypeMap:
		return fmt.Sprintf(`map[string]any{"key1": "%s-val1", "key2": "%s-val2"}`, a.FullName, a.FullName)
	case pcommon.ValueTypeSlice:
		return fmt.Sprintf(`[]any{"%s-item1", "%s-item2"}`, a.FullName, a.FullName)
	case pcommon.ValueTypeBytes:
		return fmt.Sprintf(`[]byte("%s-val")`, a.FullName)
	}
	return ""
}

type tests struct {
	Config              any  `mapstructure:"config"`
	SkipLifecycle       bool `mapstructure:"skip_lifecycle"`
	ExpectConsumerError bool `mapstructure:"expect_consumer_error"`
}

type metadata struct {
	// Type of the component.
	Type string `mapstructure:"type"`
	// Type of the parent component (applicable to subcomponents).
	Parent string `mapstructure:"parent"`
	// Status information for the component.
	Status *Status `mapstructure:"status"`
	// SemConvVersion is a version number of OpenTelemetry semantic conventions applied to the scraped metrics.
	SemConvVersion string `mapstructure:"sem_conv_version"`
	// ResourceAttributes that can be emitted by the component.
	ResourceAttributes map[attributeName]attribute `mapstructure:"resource_attributes"`
	// Attributes emitted by one or more metrics.
	Attributes map[attributeName]attribute `mapstructure:"attributes"`
	// Metrics that can be emitted by the component.
	Metrics map[metricName]metric `mapstructure:"metrics"`
	// ScopeName of the metrics emitted by the component.
	ScopeName string `mapstructure:"-"`
	// ShortFolderName is the shortened folder name of the component, removing class if present
	ShortFolderName string `mapstructure:"-"`

	Tests *tests `mapstructure:"tests"`
}

func setAttributesFullName(attrs map[attributeName]attribute) {
	for k, v := range attrs {
		v.FullName = k
		attrs[k] = v
	}
}

type templateContext struct {
	metadata
	// Package name for generated code.
	Package string
}

func loadMetadata(filePath string) (metadata, error) {
	cp, err := fileprovider.New().Retrieve(context.Background(), "file:"+filePath, nil)
	if err != nil {
		return metadata{}, err
	}

	conf, err := cp.AsConf()
	if err != nil {
		return metadata{}, err
	}

	md := metadata{ScopeName: scopeName(filePath), ShortFolderName: shortFolderName(filePath)}
	if err := conf.Unmarshal(&md, confmap.WithErrorUnused()); err != nil {
		return md, err
	}

	if err := md.Validate(); err != nil {
		return md, err
	}

	setAttributesFullName(md.Attributes)
	setAttributesFullName(md.ResourceAttributes)

	return md, nil
}

var componentTypes = map[string]func(string) string{
	"connector": func(in string) string { return strings.TrimSuffix(in, "connector") },
	"exporter":  func(in string) string { return strings.TrimSuffix(in, "exporter") },
	"extension": func(in string) string { return strings.TrimSuffix(in, "extension") },
	"processor": func(in string) string { return strings.TrimSuffix(in, "processor") },
	"scraper":   func(in string) string { return strings.TrimSuffix(in, "scraper") },
	"receiver":  func(in string) string { return in },
}

func shortFolderName(filePath string) string {
	parentFolder := filepath.Base(filepath.Dir(filePath))
	for cType := range componentTypes {
		if strings.HasSuffix(parentFolder, cType) {
			return strings.TrimSuffix(parentFolder, cType)
		}
	}
	return parentFolder
}

func scopeName(filePath string) string {
	sn := "otelcol"
	dirs := strings.Split(filepath.Dir(filePath), string(os.PathSeparator))
	for _, dir := range dirs {
		// skip directory names for component types
		if _, ok := componentTypes[dir]; ok {
			continue
		}
		// note here that the only component that receives a different
		// treatment is receivers. this is to prevent breaking backwards
		// compatibility for anyone that's using the generated metrics w/
		// scope names today.
		for cType, normalizeFunc := range componentTypes {
			if strings.HasSuffix(dir, cType) {
				sn += "/" + normalizeFunc(dir)
			}
		}
	}
	return sn
}
