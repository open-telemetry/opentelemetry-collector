// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resource // import "go.opentelemetry.io/collector/service/internal/resource"

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"

	"go.opentelemetry.io/collector/component"
)

// Config models the resource configuration defined by the declarative configuration schema.
// Unknown keys inside the resource block (legacy inline attributes) are captured in DeprecatedAttributes.
type Config struct {
	// Attributes is the new array format for resource attributes.
	Attributes []Attribute `mapstructure:"attributes"`
	// SchemaURL allows overriding the default schema URL.
	SchemaURL *string `mapstructure:"schema_url"`
	// DeprecatedAttributes captures legacy inline attributes for backward compatibility.
	// This field is deprecated and will be removed in a future release.
	DeprecatedAttributes map[string]any `mapstructure:",remain"`
}

// Attribute represents a resource attribute entry with name and value.
// Setting Value to nil removes the attribute from the resource.
type Attribute struct {
	Name  string `mapstructure:"name"`  // Attribute name (required)
	Value any    `mapstructure:"value"` // Attribute value (nil to remove attribute)
}

// Option is a functional option for resource creation.
type Option func(*options)

type options struct {
	instanceIDGenerator func() (string, error)
}

// WithInstanceIDGenerator allows overriding the instance ID generator.
// This is primarily useful for testing.
func WithInstanceIDGenerator(gen func() (string, error)) Option {
	return func(o *options) {
		o.instanceIDGenerator = gen
	}
}

// New builds a resource for collector telemetry.
// Default service.* attributes are applied when missing.
//
// The function processes attributes in the following order:
// 1. Legacy deprecated attributes (if any) - for backward compatibility
// 2. New array attributes (if any) - can override legacy
// 3. Default service attributes (service.name, service.version, service.instance.id)
//
// Attributes can be removed by setting them to nil in the configuration.
func New(buildInfo component.BuildInfo, cfg *Config, opts ...Option) (*sdkresource.Resource, error) {
	if cfg == nil {
		cfg = &Config{}
	}

	o := &options{
		instanceIDGenerator: newInstanceID,
	}
	for _, opt := range opts {
		opt(o)
	}

	attrSet := newAttributeSet()

	if len(cfg.DeprecatedAttributes) > 0 {
		legacy, err := convertLegacyAttributes(cfg.DeprecatedAttributes)
		if err != nil {
			return nil, err
		}
		attrSet.merge(legacy)
	}

	if len(cfg.Attributes) > 0 {
		attrs, err := convertAttributeEntries(cfg.Attributes)
		if err != nil {
			return nil, err
		}
		attrSet.merge(attrs)
	}

	attrSet.ensureDefault(string(semconv.ServiceNameKey), attribute.StringValue(buildInfo.Command))
	attrSet.ensureDefault(string(semconv.ServiceVersionKey), attribute.StringValue(buildInfo.Version))
	instanceID, err := o.instanceIDGenerator()
	if err != nil {
		return nil, err
	}
	attrSet.ensureDefault(string(semconv.ServiceInstanceIDKey), attribute.StringValue(instanceID))

	schemaURL := semconv.SchemaURL
	if cfg.SchemaURL != nil {
		schemaURL = *cfg.SchemaURL
	}

	return sdkresource.NewWithAttributes(schemaURL, attrSet.keyValues()...), nil
}

func newInstanceID() (string, error) {
	instanceUUID, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("failed to generate instance ID: %w", err)
	}
	return instanceUUID.String(), nil
}

// convertLegacyAttributes converts the deprecated map[string]any format to internal representation.
// Legacy format only supports string values or nil (to remove).
func convertLegacyAttributes(values map[string]any) (map[string]*attributeValue, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make(map[string]*attributeValue, len(values))
	for key, raw := range values {
		switch val := raw.(type) {
		case nil:
			out[key] = nil
		case string:
			out[key] = &attributeValue{value: attribute.StringValue(val)}
		case *string:
			if val == nil {
				out[key] = nil
			} else {
				out[key] = &attributeValue{value: attribute.StringValue(*val)}
			}
		default:
			return nil, fmt.Errorf("legacy resource attribute %q must be string or null, got %T", key, raw)
		}
	}
	return out, nil
}

// convertAttributeEntries converts the new array format to internal representation.
func convertAttributeEntries(attrs []Attribute) (map[string]*attributeValue, error) {
	out := make(map[string]*attributeValue, len(attrs))
	for idx, attr := range attrs {
		if attr.Name == "" {
			return nil, fmt.Errorf("resource.attributes[%d] is missing name", idx)
		}
		if attr.Value == nil {
			out[attr.Name] = nil
			continue
		}

		str, ok := attr.Value.(string)
		if !ok {
			return nil, fmt.Errorf("resource attribute %q: expected string value, got %T", attr.Name, attr.Value)
		}
		out[attr.Name] = &attributeValue{value: attribute.StringValue(str)}
	}
	return out, nil
}

// attributeValue wraps an OpenTelemetry attribute.Value to allow nil values
// for attribute removal.
type attributeValue struct {
	value attribute.Value
}

// attributeSet manages a collection of resource attributes, ensuring
// deterministic ordering and handling of nil values for attribute removal.
type attributeSet struct {
	values map[string]*attributeValue
}

func newAttributeSet() *attributeSet {
	return &attributeSet{values: make(map[string]*attributeValue)}
}

func (s *attributeSet) merge(values map[string]*attributeValue) {
	if len(values) == 0 {
		return
	}
	for key, val := range values {
		if val == nil {
			s.values[key] = nil
			continue
		}
		copyVal := *val
		s.values[key] = &copyVal
	}
}

func (s *attributeSet) ensureDefault(key string, val attribute.Value) {
	if _, exists := s.values[key]; exists {
		return
	}
	s.values[key] = &attributeValue{value: val}
}

func (s *attributeSet) keyValues() []attribute.KeyValue {
	kvs := make([]attribute.KeyValue, 0, len(s.values))
	for key, val := range s.values {
		if val != nil {
			kvs = append(kvs, attribute.KeyValue{
				Key:   attribute.Key(key),
				Value: val.value,
			})
		}
	}

	sort.Slice(kvs, func(i, j int) bool {
		return kvs[i].Key < kvs[j].Key
	})
	return kvs
}
