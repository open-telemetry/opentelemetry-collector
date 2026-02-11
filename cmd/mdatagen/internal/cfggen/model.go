// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfggen"

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ConfigMetadata represents a JSON schema object, draft 2020-12 (limited), with additional custom fields.
type ConfigMetadata struct {
	Schema               string                     `mapstructure:"$schema,omitempty" json:"$schema,omitempty" yaml:"$schema,omitempty"`
	ID                   string                     `mapstructure:"$id,omitempty" json:"$id,omitempty" yaml:"$id,omitempty"`
	Title                string                     `mapstructure:"title,omitempty" json:"title,omitempty" yaml:"title,omitempty"`
	Description          string                     `mapstructure:"description,omitempty" json:"description,omitempty" yaml:"description,omitempty"`
	Comment              string                     `mapstructure:"$comment,omitempty" json:"$comment,omitempty" yaml:"$comment,omitempty"`
	Type                 any                        `mapstructure:"type,omitempty" json:"type,omitempty" yaml:"type,omitempty"`
	Ref                  string                     `mapstructure:"$ref,omitempty" json:"-" yaml:"$ref,omitempty"`
	Default              any                        `mapstructure:"default,omitempty" json:"default,omitempty" yaml:"default,omitempty"`
	Examples             []any                      `mapstructure:"examples,omitempty" json:"examples,omitempty" yaml:"examples,omitempty"`
	Deprecated           bool                       `mapstructure:"deprecated,omitempty" json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Enum                 []any                      `mapstructure:"enum,omitempty" json:"enum,omitempty" yaml:"enum,omitempty"`
	Const                any                        `mapstructure:"const,omitempty" json:"const,omitempty" yaml:"const,omitempty"`
	AllOf                []*ConfigMetadata          `mapstructure:"allOf,omitempty" json:"allOf,omitempty" yaml:"allOf,omitempty"`
	Properties           map[string]*ConfigMetadata `mapstructure:"properties,omitempty" json:"properties,omitempty" yaml:"properties,omitempty"`
	AdditionalProperties any                        `mapstructure:"additionalProperties,omitempty" json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
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
	ContentMediaType     string                     `mapstructure:"contentMediaType,omitempty" json:"contentMediaType,omitempty" yaml:"contentMediaType,omitempty"`
	ContentEncoding      string                     `mapstructure:"contentEncoding,omitempty" json:"contentEncoding,omitempty" yaml:"contentEncoding,omitempty"`
	ContentSchema        *ConfigMetadata            `mapstructure:"contentSchema,omitempty" json:"contentSchema,omitempty" yaml:"contentSchema,omitempty"`
	MultipleOf           *float64                   `mapstructure:"multipleOf,omitempty" json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`
	Maximum              *float64                   `mapstructure:"maximum,omitempty" json:"maximum,omitempty" yaml:"maximum,omitempty"`
	ExclusiveMaximum     *float64                   `mapstructure:"exclusiveMaximum,omitempty" json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
	Minimum              *float64                   `mapstructure:"minimum,omitempty" json:"minimum,omitempty" yaml:"minimum,omitempty"`
	ExclusiveMinimum     *float64                   `mapstructure:"exclusiveMinimum,omitempty" json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	Defs                 map[string]*ConfigMetadata `mapstructure:"$defs,omitempty" json:"-" yaml:"$defs,omitempty"`
	// Additional custom fields
	GoType     string `mapstructure:"x-customType,omitempty" json:"-" yaml:"x-customType,omitempty"`
	IsPointer  bool   `mapstructure:"x-pointer,omitempty" json:"-" yaml:"x-pointer,omitempty"`
	IsOptional bool   `mapstructure:"x-optional,omitempty" json:"-" yaml:"x-optional,omitempty"`
}

func (md *ConfigMetadata) ToJSON() ([]byte, error) {
	return json.MarshalIndent(md, "", "  ")
}

func (md *ConfigMetadata) Validate() error {
	var errs error
	if md.Type != "object" {
		errs = errors.Join(errs, fmt.Errorf("config type must be \"object\":, got %q", md.Type))
	}
	if len(md.Properties) == 0 && len(md.AllOf) == 0 {
		errs = errors.Join(errs, errors.New("config must not be empty"))
	}
	return errs
}
