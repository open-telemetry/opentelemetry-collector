// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.opentelemetry.io/collector/filter"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

type Metadata struct {
	// Type of the component.
	Type string `mapstructure:"type"`
	// Type of the parent component (applicable to subcomponents).
	Parent string `mapstructure:"parent"`
	// Status information for the component.
	Status *Status `mapstructure:"status"`
	// The name of the package that will be generated.
	GeneratedPackageName string `mapstructure:"generated_package_name"`
	// Telemetry information for the component.
	Telemetry Telemetry `mapstructure:"telemetry"`
	// SemConvVersion is a version number of OpenTelemetry semantic conventions applied to the scraped metrics.
	SemConvVersion string `mapstructure:"sem_conv_version"`
	// ResourceAttributes that can be emitted by the component.
	ResourceAttributes map[AttributeName]Attribute `mapstructure:"resource_attributes"`
	// Attributes emitted by one or more metrics.
	Attributes map[AttributeName]Attribute `mapstructure:"attributes"`
	// Metrics that can be emitted by the component.
	Metrics map[MetricName]Metric `mapstructure:"metrics"`
	// Events that can be emitted by the component.
	Events map[EventName]Event `mapstructure:"events"`
	// GithubProject is the project where the component README lives in the format of org/repo, defaults to open-telemetry/opentelemetry-collector-contrib
	GithubProject string `mapstructure:"github_project"`
	// ScopeName of the metrics emitted by the component.
	ScopeName string `mapstructure:"scope_name"`
	// ShortFolderName is the shortened folder name of the component, removing class if present
	ShortFolderName string `mapstructure:"-"`
	// Tests is the set of tests generated with the component
	Tests Tests `mapstructure:"tests"`
}

func (md Metadata) GetCodeCovComponentID() string {
	if md.Status.CodeCovComponentID != "" {
		return md.Status.CodeCovComponentID
	}

	return strings.ReplaceAll(md.Status.Class+"_"+md.Type, "/", "_")
}

func (md *Metadata) Validate() error {
	var errs error
	if err := md.validateType(); err != nil {
		errs = errors.Join(errs, err)
	}

	if md.Parent != "" {
		if md.Status != nil {
			// status is not required for subcomponents.
			errs = errors.Join(errs, errors.New("status must be empty for subcomponents"))
		}
	} else {
		errs = errors.Join(errs, md.Status.Validate())
	}

	if err := md.validateResourceAttributes(); err != nil {
		errs = errors.Join(errs, err)
	}

	if err := md.validateMetricsAndEvents(); err != nil {
		errs = errors.Join(errs, err)
	}

	return errs
}

// typeRegexp is used to validate the type of a component.
// A type must start with an ASCII alphabetic character and
// can only contain ASCII alphanumeric characters and '_'.
// We allow '/' for subcomponents.
// This must be kept in sync with the regex in component/config.go.
var typeRegexp = regexp.MustCompile(`^[a-zA-Z][0-9a-zA-Z_]{0,62}$`)

func (md *Metadata) validateType() error {
	if md.Type == "" {
		return errors.New("missing type")
	}

	if md.Parent != "" {
		// subcomponents are allowed to have a '/' in their type.
		return nil
	}

	if !typeRegexp.MatchString(md.Type) {
		return fmt.Errorf("invalid character(s) in type %q", md.Type)
	}
	return nil
}

func (md *Metadata) validateResourceAttributes() error {
	var errs error
	for name, attr := range md.ResourceAttributes {
		if attr.Description == "" {
			errs = errors.Join(errs, fmt.Errorf("empty description for resource attribute: %v", name))
		}
		empty := ValueType{ValueType: pcommon.ValueTypeEmpty}
		if attr.Type == empty {
			errs = errors.Join(errs, fmt.Errorf("empty type for resource attribute: %v", name))
		}
	}
	return errs
}

func (md *Metadata) validateMetricsAndEvents() error {
	var errs error
	usedAttrs := map[AttributeName]bool{}
	errs = errors.Join(errs,
		validateMetrics(md.Metrics, md.Attributes, usedAttrs),
		validateMetrics(md.Telemetry.Metrics, md.Attributes, usedAttrs),
		validateEvents(md.Events, md.Attributes, usedAttrs),
		md.validateAttributes(usedAttrs))
	return errs
}

func (md *Metadata) validateAttributes(usedAttrs map[AttributeName]bool) error {
	var errs error
	unusedAttrs := make([]AttributeName, 0, len(md.Attributes))
	for attrName, attr := range md.Attributes {
		if attr.Description == "" {
			errs = errors.Join(errs, fmt.Errorf(`missing attribute description for: %v`, attrName))
		}
		empty := ValueType{ValueType: pcommon.ValueTypeEmpty}
		if attr.Type == empty {
			errs = errors.Join(errs, fmt.Errorf("empty type for attribute: %v", attrName))
		}
		if !usedAttrs[attrName] {
			unusedAttrs = append(unusedAttrs, attrName)
		}
	}
	if len(unusedAttrs) > 0 {
		errs = errors.Join(errs, fmt.Errorf("unused attributes: %v", unusedAttrs))
	}
	return errs
}

func (md *Metadata) supportsSignal(signal string) bool {
	if md.Status == nil {
		return false
	}

	for _, signals := range md.Status.Stability {
		for _, s := range signals {
			if s == signal {
				return true
			}
		}
	}

	return false
}

func validateMetrics(metrics map[MetricName]Metric, attributes map[AttributeName]Attribute, usedAttrs map[AttributeName]bool) error {
	var errs error
	for mn, m := range metrics {
		if err := m.validate(); err != nil {
			errs = errors.Join(errs, fmt.Errorf(`metric "%v": %w`, mn, err))
			continue
		}
		unknownAttrs := make([]AttributeName, 0, len(m.Attributes))
		for _, attr := range m.Attributes {
			if _, ok := attributes[attr]; ok {
				usedAttrs[attr] = true
			} else {
				unknownAttrs = append(unknownAttrs, attr)
			}
		}
		if len(unknownAttrs) > 0 {
			errs = errors.Join(errs, fmt.Errorf(`metric "%v" refers to undefined attributes: %v`, mn, unknownAttrs))
		}
	}
	return errs
}

func validateEvents(events map[EventName]Event, attributes map[AttributeName]Attribute, usedAttrs map[AttributeName]bool) error {
	var errs error
	for en, e := range events {
		if err := e.validate(); err != nil {
			errs = errors.Join(errs, fmt.Errorf(`event "%v": %w`, en, err))
			continue
		}
		unknownAttrs := make([]AttributeName, 0, len(e.Attributes))
		for _, attr := range e.Attributes {
			if _, ok := attributes[attr]; ok {
				usedAttrs[attr] = true
			} else {
				unknownAttrs = append(unknownAttrs, attr)
			}
		}
		if len(unknownAttrs) > 0 {
			errs = errors.Join(errs, fmt.Errorf(`event "%v" refers to undefined attributes: %v`, en, unknownAttrs))
		}
	}
	return errs
}

type AttributeName string

func (mn AttributeName) Render() (string, error) {
	return FormatIdentifier(string(mn), true)
}

func (mn AttributeName) RenderUnexported() (string, error) {
	return FormatIdentifier(string(mn), false)
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
	return strings.Title(strings.ToLower(mvt.ValueType.String())) //nolint:staticcheck // SA1019
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

type Warnings struct {
	// A warning that will be displayed if the field is enabled in user config.
	IfEnabled string `mapstructure:"if_enabled"`
	// A warning that will be displayed if `enabled` field is not set explicitly in user config.
	IfEnabledNotSet string `mapstructure:"if_enabled_not_set"`
	// A warning that will be displayed if the field is configured by user in any way.
	IfConfigured string `mapstructure:"if_configured"`
}

type Attribute struct {
	// Description describes the purpose of the attribute.
	Description string `mapstructure:"description"`
	// NameOverride can be used to override the attribute name.
	NameOverride string `mapstructure:"name_override"`
	// Enabled defines whether the attribute is enabled by default.
	Enabled bool `mapstructure:"enabled"`
	// Include can be used to filter attributes.
	Include []filter.Config `mapstructure:"include"`
	// Include can be used to filter attributes.
	Exclude []filter.Config `mapstructure:"exclude"`
	// Enum can optionally describe the set of values to which the attribute can belong.
	Enum []string `mapstructure:"enum"`
	// Type is an attribute type.
	Type ValueType `mapstructure:"type"`
	// FullName is the attribute name populated from the map key.
	FullName AttributeName `mapstructure:"-"`
	// Warnings that will be shown to user under specified conditions.
	Warnings Warnings `mapstructure:"warnings"`
	// Optional defines whether the attribute is required.
	Optional bool `mapstructure:"optional"`
}

// Name returns actual name of the attribute that is set on the metric after applying NameOverride.
func (a Attribute) Name() AttributeName {
	if a.NameOverride != "" {
		return AttributeName(a.NameOverride)
	}
	return a.FullName
}

func (a Attribute) TestValue() string {
	if a.Enum != nil {
		return fmt.Sprintf(`"%s"`, a.Enum[0])
	}
	switch a.Type.ValueType {
	case pcommon.ValueTypeEmpty:
		return ""
	case pcommon.ValueTypeStr:
		return fmt.Sprintf(`"%s-val"`, a.FullName)
	case pcommon.ValueTypeInt:
		return strconv.Itoa(len(a.FullName))
	case pcommon.ValueTypeDouble:
		return fmt.Sprintf("%f", 0.1+float64(len(a.FullName)))
	case pcommon.ValueTypeBool:
		return strconv.FormatBool(len(a.FullName)%2 == 0)
	case pcommon.ValueTypeMap:
		return fmt.Sprintf(`map[string]any{"key1": "%s-val1", "key2": "%s-val2"}`, a.FullName, a.FullName)
	case pcommon.ValueTypeSlice:
		return fmt.Sprintf(`[]any{"%s-item1", "%s-item2"}`, a.FullName, a.FullName)
	case pcommon.ValueTypeBytes:
		return fmt.Sprintf(`[]byte("%s-val")`, a.FullName)
	}
	return ""
}

type Signal struct {
	// Enabled defines whether the signal is enabled by default.
	Enabled bool `mapstructure:"enabled"`

	// Warnings that will be shown to user under specified conditions.
	Warnings Warnings `mapstructure:"warnings"`

	// Description of the signal.
	Description string `mapstructure:"description"`

	// The stability level of the signal.
	Stability Stability `mapstructure:"stability"`

	// Extended documentation of the signal. If specified, this will be appended to the description used in generated documentation.
	ExtendedDocumentation string `mapstructure:"extended_documentation"`

	// Attributes is the list of attributes that the signal emits.
	Attributes []AttributeName `mapstructure:"attributes"`
}

func (s Signal) HasOptionalAttribute(attrs map[AttributeName]Attribute) bool {
	for _, attr := range s.Attributes {
		if v, exists := attrs[attr]; exists && v.Optional {
			return true
		}
	}
	return false
}
