// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/filter"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

type Metadata struct {
	// Type of the component.
	Type string `mapstructure:"type"`
	// DisplayName is a human-readable display name for the component.
	DisplayName string `mapstructure:"display_name"`
	// Description is a brief description of the component.
	Description string `mapstructure:"description"`
	// Type of the parent component (applicable to subcomponents).
	Parent string `mapstructure:"parent"`
	// Status information for the component.
	Status *Status `mapstructure:"status"`
	// Spatial Re-aggregation featuregate.
	ReaggregationEnabled bool `mapstructure:"reaggregation_enabled"`
	// The name of the package that will be generated.
	GeneratedPackageName string `mapstructure:"generated_package_name"`
	// Telemetry information for the component.
	Telemetry Telemetry `mapstructure:"telemetry"`
	// SemConvVersion is a version number of OpenTelemetry semantic conventions applied to the scraped metrics.
	SemConvVersion string `mapstructure:"sem_conv_version"`
	// ResourceAttributes that can be emitted by the component.
	ResourceAttributes map[AttributeName]Attribute `mapstructure:"resource_attributes"`
	// Entities organizes resource attributes into logical entities.
	Entities []Entity `mapstructure:"entities"`
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
	// PackageName is the name of the package where the component is defined.
	PackageName string `mapstructure:"package_name"`
	// FeatureGates that are managed by the component.
	FeatureGates []FeatureGate `mapstructure:"feature_gates"`
}

type Deprecated struct {
	Since string `mapstructure:"since"`
	Note  string `mapstructure:"note"`
}

func (d *Deprecated) validate() error {
	if strings.TrimSpace(d.Since) == "" {
		return errors.New("deprecated.since must be set")
	}

	// NOTE: note is optional, but if present, it must not be empty
	if d.Note != "" && strings.TrimSpace(d.Note) == "" {
		return errors.New("deprecated.note must not be empty")
	}

	return nil
}

func (md Metadata) GetCodeCovComponentID() string {
	if md.Status.CodeCovComponentID != "" {
		return md.Status.CodeCovComponentID
	}

	return strings.ReplaceAll(md.Status.Class+"_"+md.Type, "/", "_")
}

func (md Metadata) HasEntities() bool {
	return len(md.Entities) > 0
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

	if err := md.validateEntities(); err != nil {
		errs = errors.Join(errs, err)
	}

	if err := md.validateMetricsAndEvents(); err != nil {
		errs = errors.Join(errs, err)
	}

	if err := md.validateFeatureGates(); err != nil {
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
		if attr.EnabledPtr == nil {
			errs = errors.Join(errs, fmt.Errorf("enabled field is required for resource attribute: %v", name))
		}
	}
	return errs
}

func (md *Metadata) validateEntities() error {
	var errs error
	usedAttrs := make(map[AttributeName]string)
	seenTypes := make(map[string]bool)

	// First pass: collect entity types and validate basic entity properties
	for _, entity := range md.Entities {
		if entity.Type == "" {
			errs = errors.Join(errs, errors.New("entity type cannot be empty"))
			continue
		}
		if seenTypes[entity.Type] {
			errs = errors.Join(errs, fmt.Errorf(`duplicate entity type: %v`, entity.Type))
		}
		seenTypes[entity.Type] = true

		if entity.Brief == "" {
			errs = errors.Join(errs, fmt.Errorf(`entity "%v": brief is required`, entity.Type))
		}
		if len(entity.Identity) == 0 {
			errs = errors.Join(errs, fmt.Errorf(`entity "%v": identity is required`, entity.Type))
		}
		for _, ref := range entity.Identity {
			if _, ok := md.ResourceAttributes[ref.Ref]; !ok {
				errs = errors.Join(errs, fmt.Errorf(`entity "%v": identity refers to undefined resource attribute: %v`, entity.Type, ref.Ref))
			}
			if otherEntity, used := usedAttrs[ref.Ref]; used {
				errs = errors.Join(errs, fmt.Errorf(`entity "%v": attribute %v is already used by entity "%v"`, entity.Type, ref.Ref, otherEntity))
			} else {
				usedAttrs[ref.Ref] = entity.Type
			}
		}
		for _, ref := range entity.Description {
			if _, ok := md.ResourceAttributes[ref.Ref]; !ok {
				errs = errors.Join(errs, fmt.Errorf(`entity "%v": description refers to undefined resource attribute: %v`, entity.Type, ref.Ref))
			}
			if otherEntity, used := usedAttrs[ref.Ref]; used {
				errs = errors.Join(errs, fmt.Errorf(`entity "%v": attribute %v is already used by entity "%v"`, entity.Type, ref.Ref, otherEntity))
			} else {
				usedAttrs[ref.Ref] = entity.Type
			}
		}
	}

	// Second pass: validate relationships
	seenRelationships := make(map[string]string)
	for _, entity := range md.Entities {
		for _, rel := range entity.Relationships {
			if rel.Type == "" {
				errs = errors.Join(errs, fmt.Errorf(`entity "%v": relationship type cannot be empty`, entity.Type))
				continue
			}
			if rel.Target == "" {
				errs = errors.Join(errs, fmt.Errorf(`entity "%v": relationship target cannot be empty`, entity.Type))
				continue
			}
			if !seenTypes[rel.Target] {
				errs = errors.Join(errs, fmt.Errorf(`entity "%v": relationship target "%v" does not exist`, entity.Type, rel.Target))
				continue
			}
			if seenRelationships[rel.Target] == entity.Type || seenRelationships[entity.Type] == rel.Target {
				errs = errors.Join(errs, fmt.Errorf(`entity "%v": duplicate relationship to target "%v" (only one relationship allowed between two entities)`, entity.Type, rel.Target))
				continue
			}
			seenRelationships[rel.Target] = entity.Type
			seenRelationships[entity.Type] = rel.Target
		}
	}
	return errs
}

func (md *Metadata) validateMetricsAndEvents() error {
	var errs error
	usedAttrs := map[AttributeName]bool{}
	errs = errors.Join(errs,
		validateMetrics(md.Metrics, md.Attributes, usedAttrs, md.SemConvVersion),
		validateMetrics(md.Telemetry.Metrics, md.Attributes, usedAttrs, md.SemConvVersion),
		validateEvents(md.Events, md.Attributes, usedAttrs),
		md.validateAttributes(usedAttrs),
		md.validateEntityAssociations())
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
		if attr.EnabledPtr != nil {
			errs = errors.Join(errs, fmt.Errorf("enabled field is not allowed for regular attribute: %v", attrName))
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

// validateEntityAssociations checks that if entities are defined, then each metric and event must be associated with an entity.
func (md *Metadata) validateEntityAssociations() error {
	var errs error
	requireEntityAssociation := len(md.Entities) > 0
	entityTypes := make(map[string]bool)
	for _, entity := range md.Entities {
		entityTypes[entity.Type] = true
	}

	for metricName, metric := range md.Metrics {
		if requireEntityAssociation && metric.Entity == "" {
			errs = errors.Join(errs, fmt.Errorf(`metric "%v": entity is required when entities are defined`, metricName))
		}
		if metric.Entity != "" && !entityTypes[metric.Entity] {
			errs = errors.Join(errs, fmt.Errorf(`metric "%v": entity refers to undefined entity type: %v`, metricName, metric.Entity))
		}
	}

	for eventName, event := range md.Events {
		if requireEntityAssociation && event.Entity == "" {
			errs = errors.Join(errs, fmt.Errorf(`event "%v": entity is required when entities are defined`, eventName))
		}
		if event.Entity != "" && !entityTypes[event.Entity] {
			errs = errors.Join(errs, fmt.Errorf(`event "%v": entity refers to undefined entity type: %v`, eventName, event.Entity))
		}
	}

	return errs
}

func (md *Metadata) supportsSignal(signal string) bool {
	if md.Status == nil {
		return false
	}

	for _, signals := range md.Status.Stability {
		if slices.Contains(signals, signal) {
			return true
		}
	}

	return false
}

func validateMetrics(metrics map[MetricName]Metric, attributes map[AttributeName]Attribute, usedAttrs map[AttributeName]bool, semConvVersion string) error {
	var errs error
	for mn, m := range metrics {
		if err := m.validate(mn, semConvVersion); err != nil {
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

func (md *Metadata) validateFeatureGates() error {
	var errs error
	seen := make(map[FeatureGateID]bool)
	idRegexp := regexp.MustCompile(`^[0-9a-zA-Z.]*$`)

	// Validate that feature gates are sorted by ID
	if !slices.IsSortedFunc(md.FeatureGates, func(a, b FeatureGate) int {
		return strings.Compare(string(a.ID), string(b.ID))
	}) {
		errs = errors.Join(errs, errors.New("feature gates must be sorted by ID"))
	}

	for i, gate := range md.FeatureGates {
		// Validate gate ID is not empty
		if string(gate.ID) == "" {
			errs = errors.Join(errs, fmt.Errorf("feature gate at index %d: ID cannot be empty", i))
			continue
		}

		// Validate ID follows the allowed character pattern
		if !idRegexp.MatchString(string(gate.ID)) {
			errs = errors.Join(errs, fmt.Errorf(`feature gate "%v": ID contains invalid characters, must match ^[0-9a-zA-Z.]*$`, gate.ID))
		}

		// Check for duplicate IDs
		if seen[gate.ID] {
			errs = errors.Join(errs, fmt.Errorf(`feature gate "%v": duplicate ID`, gate.ID))
			continue
		}
		seen[gate.ID] = true

		// Validate gate has required fields
		if gate.Description == "" {
			errs = errors.Join(errs, fmt.Errorf(`feature gate "%v": description is required`, gate.ID))
		}

		// Validate that each feature gate has a reference link
		if gate.ReferenceURL == "" {
			errs = errors.Join(errs, fmt.Errorf(`feature gate "%v": reference_url is required`, gate.ID))
		}

		// Validate stage is one of the allowed values
		validStages := map[FeatureGateStage]bool{
			FeatureGateStageAlpha:      true,
			FeatureGateStageBeta:       true,
			FeatureGateStageStable:     true,
			FeatureGateStageDeprecated: true,
		}
		if !validStages[gate.Stage] {
			errs = errors.Join(errs, fmt.Errorf(`feature gate "%v": invalid stage "%v", must be one of: alpha, beta, stable, deprecated`, gate.ID, gate.Stage))
		}

		// Validate from_version is required
		if gate.FromVersion == "" {
			errs = errors.Join(errs, fmt.Errorf(`feature gate "%v": from_version is required`, gate.ID))
		} else if !strings.HasPrefix(gate.FromVersion, "v") {
			errs = errors.Join(errs, fmt.Errorf(`feature gate "%v": from_version "%v" must start with 'v'`, gate.ID, gate.FromVersion))
		}
		if gate.ToVersion != "" && !strings.HasPrefix(gate.ToVersion, "v") {
			errs = errors.Join(errs, fmt.Errorf(`feature gate "%v": to_version "%v" must start with 'v'`, gate.ID, gate.ToVersion))
		}

		// Validate that stable/deprecated gates should have to_version
		if (gate.Stage == FeatureGateStageStable || gate.Stage == FeatureGateStageDeprecated) && gate.ToVersion == "" {
			errs = errors.Join(errs, fmt.Errorf(`feature gate "%v": to_version is required for %v stage gates`, gate.ID, gate.Stage))
		}
	}
	return errs
}

type AttributeName string

// AttributeRequirementLevel defines the requirement level of an attribute.
type AttributeRequirementLevel string

const (
	// AttributeRequirementLevelRequired means the attribute is always included and cannot be excluded.
	AttributeRequirementLevelRequired AttributeRequirementLevel = "required"
	// AttributeRequirementLevelConditionallyRequired means the attribute is included by default when certain conditions are met.
	AttributeRequirementLevelConditionallyRequired AttributeRequirementLevel = "conditionally_required"
	// AttributeRequirementLevelRecommended means the attribute is included by default but can be disabled via configuration.
	AttributeRequirementLevelRecommended AttributeRequirementLevel = "recommended"
	// AttributeRequirementLevelOptIn means the attribute is not included unless explicitly enabled in user config.
	AttributeRequirementLevelOptIn AttributeRequirementLevel = "opt_in"
)

// String returns capitalized display name of the requirement level for documentation.
func (rl AttributeRequirementLevel) String() string {
	switch rl {
	case AttributeRequirementLevelRequired:
		return "Required"
	case AttributeRequirementLevelConditionallyRequired:
		return "Conditionally Required"
	case AttributeRequirementLevelRecommended:
		return "Recommended"
	case AttributeRequirementLevelOptIn:
		return "Opt-In"
	}
	return ""
}

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

type SemanticConvention struct {
	SemanticConventionRef string `mapstructure:"ref"`
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
	// EnabledPtr defines whether the attribute is enabled by default.
	EnabledPtr *bool `mapstructure:"enabled"`
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
	// RequirementLevel defines the requirement level of the attribute.
	RequirementLevel AttributeRequirementLevel `mapstructure:"requirement_level"`
}

// IsConditional returns true if the attribute is conditionally required.
func (a Attribute) IsConditional() bool {
	return a.RequirementLevel == AttributeRequirementLevelConditionallyRequired
}

// IsRequired returns true if the attribute is required.
func (a Attribute) IsRequired() bool {
	return a.RequirementLevel == AttributeRequirementLevelRequired
}

// IsNotOptIn returns true if the attribute is any requirement_level above
// opt_in
func (a Attribute) IsNotOptIn() bool {
	return a.RequirementLevel != AttributeRequirementLevelOptIn
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (rl *AttributeRequirementLevel) UnmarshalText(text []byte) error {
	switch string(text) {
	case "required":
		*rl = AttributeRequirementLevelRequired
	case "conditionally_required":
		*rl = AttributeRequirementLevelConditionallyRequired
	case "recommended":
		*rl = AttributeRequirementLevelRecommended
	case "opt_in":
		*rl = AttributeRequirementLevelOptIn
	case "":
		*rl = AttributeRequirementLevelRecommended
	default:
		return fmt.Errorf("invalid requirement_level %q", string(text))
	}
	return nil
}

// Enabled returns the boolean value of EnabledPtr.
// This method is needed to differentiate between different types of attributes:
// - Resource attributes: EnabledPtr is always set (non-nil) due to validation
// - Regular attributes: EnabledPtr is always nil due to validation
// Panics if EnabledPtr is nil, indicating incorrect template usage.
func (a Attribute) Enabled() bool {
	if a.EnabledPtr == nil {
		panic("Enabled() must not be called on regular attributes, only on resource attributes")
	}
	return *a.EnabledPtr
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
		return fmt.Sprintf(`%q`, a.Enum[0])
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

func (a Attribute) TestValueTwo() string {
	if a.Enum != nil {
		return fmt.Sprintf(`%q`, a.Enum[1])
	}
	switch a.Type.ValueType {
	case pcommon.ValueTypeEmpty:
		return ""
	case pcommon.ValueTypeStr:
		return fmt.Sprintf(`"%s-val-2"`, a.FullName)
	case pcommon.ValueTypeInt:
		return strconv.Itoa(len(a.FullName) + 1)
	case pcommon.ValueTypeDouble:
		return fmt.Sprintf("%f", 1.1+float64(len(a.FullName)))
	case pcommon.ValueTypeBool:
		return strconv.FormatBool(len(a.FullName)%2 == 1)
	case pcommon.ValueTypeMap:
		return fmt.Sprintf(`map[string]any{"key3": "%s-val3", "key4": "%s-val4"}`, a.FullName, a.FullName)
	case pcommon.ValueTypeSlice:
		return fmt.Sprintf(`[]any{"%s-item3", "%s-item4"}`, a.FullName, a.FullName)
	case pcommon.ValueTypeBytes:
		return fmt.Sprintf(`[]byte("%s-val-2")`, a.FullName)
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

	// The semantic convention reference of the signal.
	SemanticConvention *SemanticConvention `mapstructure:"semantic_convention"`

	// The stability level of the signal.
	Stability component.StabilityLevel `mapstructure:"stability"`

	// Extended documentation of the signal. If specified, this will be appended to the description used in generated documentation.
	ExtendedDocumentation string `mapstructure:"extended_documentation"`

	// Attributes is the list of attributes that the signal emits.
	Attributes []AttributeName `mapstructure:"attributes"`

	// Entity is the type of entity this signal is associated with.
	// Required when entities are defined.
	Entity string `mapstructure:"entity"`
}

func (s Signal) HasConditionalAttributes(attrs map[AttributeName]Attribute) bool {
	for _, attr := range s.Attributes {
		if v, exists := attrs[attr]; exists && v.IsConditional() {
			return true
		}
	}
	return false
}

type Entity struct {
	// Type is the type of the entity.
	Type string `mapstructure:"type"`
	// Brief is a brief description of the entity.
	Brief string `mapstructure:"brief"`
	// Stability is the stability level of the entity.
	Stability component.StabilityLevel `mapstructure:"stability"`
	// Identity contains references to resource attributes that uniquely identify the entity.
	Identity []EntityAttributeRef `mapstructure:"identity"`
	// Description contains references to resource attributes that describe the entity.
	Description []EntityAttributeRef `mapstructure:"description"`
	// Relationships defines how this entity relates to other entities (optional).
	// Relationships should be defined only on one end. It is recommended to define
	// relationships on entities with lower lifespan (higher churn).
	Relationships []EntityRelationship `mapstructure:"relationships"`
}

type EntityAttributeRef struct {
	// Ref is the reference to a resource attribute.
	Ref AttributeName `mapstructure:"ref"`
}

type EntityRelationship struct {
	// Type is the relationship type (e.g., "parent", "child", "peer").
	Type string `mapstructure:"type"`
	// Target is the entity type this entity relates to.
	Target string `mapstructure:"target"`
}

// FeatureGateID represents the identifier for a feature gate.
type FeatureGateID string

// FeatureGateStage represents the lifecycle stage of a feature gate.
type FeatureGateStage string

const (
	FeatureGateStageAlpha      FeatureGateStage = "alpha"
	FeatureGateStageBeta       FeatureGateStage = "beta"
	FeatureGateStageStable     FeatureGateStage = "stable"
	FeatureGateStageDeprecated FeatureGateStage = "deprecated"
)

// FeatureGate represents a feature gate definition in metadata.
type FeatureGate struct {
	// ID is the unique identifier for the feature gate.
	ID FeatureGateID `mapstructure:"id"`
	// Description of the feature gate.
	Description string `mapstructure:"description"`
	// Stage is the lifecycle stage of the feature gate.
	Stage FeatureGateStage `mapstructure:"stage"`
	// FromVersion is the version when the feature gate was introduced.
	FromVersion string `mapstructure:"from_version"`
	// ToVersion is the version when the feature gate reached stable stage.
	ToVersion string `mapstructure:"to_version"`
	// ReferenceURL is the URL with contextual information about the feature gate.
	ReferenceURL string `mapstructure:"reference_url"`
}
