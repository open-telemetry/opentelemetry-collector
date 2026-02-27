// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"
	"errors"

	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	internalresource "go.opentelemetry.io/collector/service/internal/resource"
	"go.opentelemetry.io/collector/service/telemetry"
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry/internal/migration"
)

var defaultAttributeValues = internalresource.DefaultAttributeValues

func createResource(
	_ context.Context,
	set telemetry.Settings,
	componentConfig component.Config,
) (pcommon.Resource, error) {
	cfg := componentConfig.(*Config)
	resCfg, err := resourceConfigWithDefaults(set.BuildInfo, &cfg.Resource)
	if err != nil {
		return pcommon.Resource{}, err
	}
	return pcommonResourceFromConfig(resCfg)
}

// pcommonAttrsToOTelAttrs gets the Resource attributes to OpenTelemetry attribute.KeyValue slice.
func pcommonAttrsToOTelAttrs(resource *pcommon.Resource) []attribute.KeyValue {
	var result []attribute.KeyValue
	if resource != nil {
		attrs := resource.Attributes()
		attrs.Range(func(k string, v pcommon.Value) bool {
			result = append(result, pcommonValueToAttribute(k, v))
			return true
		})
	}
	return result
}

func resourceConfigWithDefaults(buildInfo component.BuildInfo, cfg *migration.ResourceConfigV030) (config.Resource, error) {
	if cfg == nil {
		cfg = &migration.ResourceConfigV030{}
	}

	// NOTE: this function does not mutate cfg; it builds a derived resource config.

	// Resource detectors are accepted in config for forward compatibility,
	// but are not yet applied to build the resource.
	resourceCfg := cfg.Resource
	resourceCfg.Attributes = append([]config.AttributeNameValue(nil), cfg.Attributes...)

	if resourceCfg.SchemaUrl == nil {
		resourceCfg.SchemaUrl = ptr(semconv.SchemaURL)
		if cfg.Resource.SchemaUrl == nil {
			cfg.Resource.SchemaUrl = resourceCfg.SchemaUrl
		}
	}

	seen := make(map[string]struct{}, len(resourceCfg.Attributes))
	explicitlyRemoved := make(map[string]struct{})
	for _, attr := range resourceCfg.Attributes {
		if attr.Name == "" {
			return config.Resource{}, errors.New("resource attribute is missing name")
		}
		seen[attr.Name] = struct{}{}
		if attr.Value == nil {
			explicitlyRemoved[attr.Name] = struct{}{}
		}
	}

	attrsList, err := parseAttributesList(resourceCfg.AttributesList)
	if err != nil {
		return config.Resource{}, err
	}
	for _, attr := range attrsList {
		if attr.Name == "" {
			return config.Resource{}, errors.New("resource attribute is missing name")
		}
		seen[attr.Name] = struct{}{}
	}

	removedDefaults := make(map[string]struct{})
	for _, name := range []string{
		string(semconv.ServiceNameKey),
		string(semconv.ServiceVersionKey),
		string(semconv.ServiceInstanceIDKey),
	} {
		if cfg.IsRemoved(name) {
			removedDefaults[name] = struct{}{}
			continue
		}
		if _, ok := explicitlyRemoved[name]; ok {
			removedDefaults[name] = struct{}{}
		}
	}
	defaults, err := defaultAttributeValues(buildInfo, removedDefaults)
	if err != nil {
		return config.Resource{}, err
	}
	for _, name := range []string{
		string(semconv.ServiceNameKey),
		string(semconv.ServiceVersionKey),
		string(semconv.ServiceInstanceIDKey),
	} {
		if _, ok := seen[name]; ok {
			continue
		}
		if value, ok := defaults[name]; ok {
			attr := config.AttributeNameValue{
				Name:  name,
				Value: value,
			}
			resourceCfg.Attributes = append(resourceCfg.Attributes, attr)
			cfg.Resource.Attributes = append(cfg.Resource.Attributes, attr)
			seen[name] = struct{}{}
		}
	}

	filtered := make([]config.AttributeNameValue, 0, len(resourceCfg.Attributes))
	for _, attr := range resourceCfg.Attributes {
		if attr.Value == nil {
			continue
		}
		filtered = append(filtered, attr)
	}

	return config.Resource{
		Attributes:     filtered,
		AttributesList: resourceCfg.AttributesList,
		Detectors:      resourceCfg.Detectors,
		SchemaUrl:      resourceCfg.SchemaUrl,
	}, nil
}
