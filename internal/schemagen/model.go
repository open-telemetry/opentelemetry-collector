// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/internal/schemagen"

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"go.opentelemetry.io/collector/confmap"
)

// ConfigMetadata represents a component's configuration definition of metadata.yaml
type ConfigMetadata struct {
	Description          string                     `mapstructure:"description,omitempty" json:"description,omitempty" yaml:"description,omitempty"`
	Type                 string                     `mapstructure:"type,omitempty" json:"type,omitempty" yaml:"type,omitempty"`
	Ref                  string                     `mapstructure:"$ref,omitempty" json:"-" yaml:"$ref,omitempty"`
	Default              any                        `mapstructure:"default,omitempty" json:"default,omitempty" yaml:"default,omitempty"`
	Deprecated           bool                       `mapstructure:"deprecated,omitempty" json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Enum                 []any                      `mapstructure:"enum,omitempty" json:"enum,omitempty" yaml:"enum,omitempty"`
	Properties           map[string]*ConfigMetadata `mapstructure:"properties,omitempty" json:"properties,omitempty" yaml:"properties,omitempty"`
	AdditionalProperties *ConfigMetadata            `mapstructure:"additionalProperties,omitempty" json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
	Required             []string                   `mapstructure:"required,omitempty" json:"required,omitempty" yaml:"required,omitempty"`
	MinProperties        *int                       `mapstructure:"minProperties,omitempty" json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	MaxProperties        *int                       `mapstructure:"maxProperties,omitempty" json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
	Items                *ConfigMetadata            `mapstructure:"items,omitempty" json:"items,omitempty" yaml:"items,omitempty"`
	MinItems             *int                       `mapstructure:"minItems,omitempty" json:"minItems,omitempty" yaml:"minItems,omitempty"`
	MaxItems             *int                       `mapstructure:"maxItems,omitempty" json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	UniqueItems          bool                       `mapstructure:"uniqueItems,omitempty" json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`
	MaxLength            *int                       `mapstructure:"maxLength,omitempty" json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	MinLength            *int                       `mapstructure:"minLength,omitempty" json:"minLength,omitempty" yaml:"minLength,omitempty"`
	Pattern              string                     `mapstructure:"pattern,omitempty" json:"pattern,omitempty" yaml:"pattern,omitempty"`
	Format               string                     `mapstructure:"format,omitempty" json:"format,omitempty" yaml:"format,omitempty"`
	Maximum              *float64                   `mapstructure:"maximum,omitempty" json:"maximum,omitempty" yaml:"maximum,omitempty"`
	ExclusiveMaximum     *float64                   `mapstructure:"exclusiveMaximum,omitempty" json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
	Minimum              *float64                   `mapstructure:"minimum,omitempty" json:"minimum,omitempty" yaml:"minimum,omitempty"`
	ExclusiveMinimum     *float64                   `mapstructure:"exclusiveMinimum,omitempty" json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	// Additional custom fields
	GoStruct   GoStructConfig `mapstructure:"go_struct,omitempty" json:"-" yaml:"go_struct,omitempty"`
	GoType     string         `mapstructure:"x-customType,omitempty" json:"-" yaml:"x-customType,omitempty"`
	IsPointer  bool           `mapstructure:"x-pointer,omitempty" json:"-" yaml:"x-pointer,omitempty"`
	IsOptional bool           `mapstructure:"x-optional,omitempty" json:"-" yaml:"x-optional,omitempty"`
	Embed      bool           `mapstructure:"embed,omitempty" json:"-" yaml:"embed,omitempty"`
	// internal
	InternalOnly bool `mapstructure:"-" json:"-" yaml:"-"`
}

type ConfigsMetadata struct {
	Config          *ConfigMetadata            `mapstructure:"config,omitempty" json:"config,omitempty" yaml:"config,omitempty"`
	ExportedConfigs map[string]*ConfigMetadata `mapstructure:"exported_configs,omitempty" json:"exported_configs,omitempty" yaml:"exported_configs,omitempty"`
}

type GoStructConfig struct {
	CustomValidator *CustomValidatorConfig `mapstructure:"custom_validator" json:"-" yaml:"custom_validator,omitempty"`
	Anonymous       bool                   `mapstructure:"anonymous" json:"-" yaml:"anonymous,omitempty"`
	IgnoreDefault   bool                   `mapstructure:"ignore_default" json:"-" yaml:"ignore_default,omitempty"`
	FieldName       string                 `mapstructure:"field_name" json:"-" yaml:"field_name,omitempty"`
}

type CustomValidatorConfig struct {
	Name string `mapstructure:"name,omitempty" json:"-" yaml:"name,omitempty"`
}

func (g *GoStructConfig) Unmarshal(parser *confmap.Conf) error {
	type goStructConfig GoStructConfig
	if err := parser.Unmarshal((*goStructConfig)(g), confmap.WithIgnoreUnused()); err != nil {
		return err
	}
	if !parser.IsSet("custom_validator") || g.CustomValidator != nil {
		return nil
	}
	sub, err := parser.Sub("custom_validator")
	if err != nil {
		return fmt.Errorf("invalid custom_validator: %w", err)
	}
	g.CustomValidator = &CustomValidatorConfig{}
	return sub.Unmarshal(g.CustomValidator)
}

func (md *ConfigsMetadata) Validate() error {
	if md.Config != nil {
		if md.Config.Type != "object" && md.Config.Type != "" {
			return fmt.Errorf("config type must be \"object\", got %q", md.Config.Type)
		}
		if err := md.Config.Validate(); err != nil {
			return err
		}
	}
	if md.ExportedConfigs != nil {
		if len(md.ExportedConfigs) == 0 {
			return errors.New("empty exported_configs section")
		}
		for _, config := range md.ExportedConfigs {
			if err := config.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

// CollectMissingDescriptions returns the config fields without a description.
func (md *ConfigsMetadata) CollectMissingDescriptions() []string {
	var missing []string
	if md.Config != nil {
		collectMissingDescriptions(md.Config, "config", &missing)
	}
	for _, name := range sortedConfigKeys(md.ExportedConfigs) {
		cfg := md.ExportedConfigs[name]
		if cfg == nil || cfg.InternalOnly {
			continue
		}
		collectMissingDescriptions(cfg, "exported_configs."+name, &missing)
	}
	sort.Strings(missing)
	return missing
}

func collectMissingDescriptions(md *ConfigMetadata, path string, missing *[]string) {
	for _, name := range sortedConfigKeys(md.Properties) {
		prop := md.Properties[name]
		if prop == nil {
			continue
		}
		fieldPath := path + "." + name
		if strings.TrimSpace(prop.Description) == "" {
			*missing = append(*missing, fieldPath)
		}
		// Ignore external types.
		if prop.Ref == "" {
			collectMissingDescriptions(prop, fieldPath, missing)
		}
	}
	if md.Items != nil && md.Items.Ref == "" {
		collectMissingDescriptions(md.Items, path+".items", missing)
	}
	if md.AdditionalProperties != nil && md.AdditionalProperties.Ref == "" {
		collectMissingDescriptions(md.AdditionalProperties, path+".additionalProperties", missing)
	}
}

func sortedConfigKeys(m map[string]*ConfigMetadata) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// MergeFrom copies any field from other that is not already set on md.
// For maps (Properties, PatternProperties), missing keys are merged in individually.
// Calling MergeFrom on a zero-value ConfigMetadata is equivalent to a deep clone of other.
func (md *ConfigMetadata) MergeFrom(other *ConfigMetadata) {
	if other == nil {
		return
	}

	// strings
	if md.Description == "" {
		md.Description = other.Description
	}
	if md.Type == "" {
		md.Type = other.Type
	}
	if md.Ref == "" {
		md.Ref = other.Ref
	}
	if md.Pattern == "" {
		md.Pattern = other.Pattern
	}
	if md.Format == "" {
		md.Format = other.Format
	}
	if md.GoType == "" {
		md.GoType = other.GoType
	}

	// any
	if md.Default == nil {
		md.Default = cloneAny(other.Default)
	}

	// booleans — false is treated as "not set"
	if !md.Deprecated {
		md.Deprecated = other.Deprecated
	}
	if !md.UniqueItems {
		md.UniqueItems = other.UniqueItems
	}
	if !md.IsPointer {
		md.IsPointer = other.IsPointer
	}
	if !md.IsOptional {
		md.IsOptional = other.IsOptional
	}
	if !md.Embed {
		md.Embed = other.Embed
	}
	if !md.InternalOnly {
		md.InternalOnly = other.InternalOnly
	}

	// slices — only fill if nil on md (no partial merge)
	if md.Enum == nil && other.Enum != nil {
		md.Enum = make([]any, len(other.Enum))
		for i, v := range other.Enum {
			md.Enum[i] = cloneAny(v)
		}
	}
	if md.Required == nil && other.Required != nil {
		md.Required = make([]string, len(other.Required))
		copy(md.Required, other.Required)
	}
	// maps — merge missing keys individually
	if other.Properties != nil {
		if md.Properties == nil {
			md.Properties = make(map[string]*ConfigMetadata, len(other.Properties))
		}
		for k, v := range other.Properties {
			if _, exists := md.Properties[k]; !exists {
				md.Properties[k] = v.Clone()
			}
		}
	}
	// *ConfigMetadata
	if md.AdditionalProperties == nil {
		md.AdditionalProperties = other.AdditionalProperties.Clone()
	}
	if md.Items == nil {
		md.Items = other.Items.Clone()
	}

	// *int
	if md.MinProperties == nil {
		md.MinProperties = clonePtr(other.MinProperties)
	}
	if md.MaxProperties == nil {
		md.MaxProperties = clonePtr(other.MaxProperties)
	}
	if md.MinItems == nil {
		md.MinItems = clonePtr(other.MinItems)
	}
	if md.MaxItems == nil {
		md.MaxItems = clonePtr(other.MaxItems)
	}
	if md.MaxLength == nil {
		md.MaxLength = clonePtr(other.MaxLength)
	}
	if md.MinLength == nil {
		md.MinLength = clonePtr(other.MinLength)
	}

	// *float64
	if md.Maximum == nil {
		md.Maximum = clonePtr(other.Maximum)
	}
	if md.ExclusiveMaximum == nil {
		md.ExclusiveMaximum = clonePtr(other.ExclusiveMaximum)
	}
	if md.Minimum == nil {
		md.Minimum = clonePtr(other.Minimum)
	}
	if md.ExclusiveMinimum == nil {
		md.ExclusiveMinimum = clonePtr(other.ExclusiveMinimum)
	}

	// GoStructConfig — merge field by field
	if md.GoStruct.CustomValidator == nil && other.GoStruct.CustomValidator != nil {
		cv := *other.GoStruct.CustomValidator
		md.GoStruct.CustomValidator = &cv
	}
	if !md.GoStruct.Anonymous {
		md.GoStruct.Anonymous = other.GoStruct.Anonymous
	}
	if !md.GoStruct.IgnoreDefault {
		md.GoStruct.IgnoreDefault = other.GoStruct.IgnoreDefault
	}
	if md.GoStruct.FieldName == "" {
		md.GoStruct.FieldName = other.GoStruct.FieldName
	}
}

func (md *ConfigMetadata) Clone() *ConfigMetadata {
	if md == nil {
		return nil
	}
	clone := &ConfigMetadata{}
	clone.MergeFrom(md)
	return clone
}

func (md *ConfigsMetadata) Clone() *ConfigsMetadata {
	if md == nil {
		return nil
	}
	clone := &ConfigsMetadata{
		Config: md.Config.Clone(),
	}
	if md.ExportedConfigs != nil {
		clone.ExportedConfigs = make(map[string]*ConfigMetadata, len(md.ExportedConfigs))
		for k, v := range md.ExportedConfigs {
			clone.ExportedConfigs[k] = v.Clone()
		}
	}
	return clone
}

func clonePtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func cloneAny(v any) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, v2 := range val {
			result[k] = cloneAny(v2)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v2 := range val {
			result[i] = cloneAny(v2)
		}
		return result
	default:
		return val
	}
}

func (md *ConfigMetadata) Validate() error {
	var errs error

	switch md.Type {
	case "object":
	case "":
		if len(md.Properties) == 0 && md.AdditionalProperties == nil && md.Ref == "" {
			errs = errors.Join(errs, errors.New("config must specify at least one property"))
		}
	}
	if len(md.Enum) > 0 && (md.Type == "object" || md.Type == "array") {
		errs = errors.Join(errs, fmt.Errorf("enum is not supported for type %q", md.Type))
	}
	for name, prop := range md.Properties {
		if err := prop.Validate(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("property %q is invalid: %w", name, err))
		}
	}
	if md.AdditionalProperties != nil {
		if err := md.AdditionalProperties.Validate(); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	if md.Items != nil {
		if err := md.Items.Validate(); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}
