// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"
	"sort"

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

	// Resource detectors are accepted in config for forward compatibility,
	// but are not yet applied to build the resource.
	resourceCfg := config.Resource{
		Attributes:     nil,
		AttributesList: cfg.AttributesList,
		Detectors:      cfg.Detectors,
		SchemaUrl:      cfg.SchemaUrl,
	}

	if resourceCfg.SchemaUrl == nil {
		resourceCfg.SchemaUrl = ptr(semconv.SchemaURL)
	}

	removedDefaults := make(map[string]struct{})
	for key, value := range cfg.LegacyAttributes {
		if value == nil {
			removedDefaults[key] = struct{}{}
		}
	}
	defaults, err := defaultAttributeValues(buildInfo, removedDefaults)
	if err != nil {
		return config.Resource{}, err
	}
	// Attribute order matters: later entries overwrite earlier ones in the SDK.
	// We rely on this behavior to prioritize declarative attributes over legacy ones,
	// and legacy ones over defaults.
	for _, name := range []string{
		string(semconv.ServiceNameKey),
		string(semconv.ServiceVersionKey),
		string(semconv.ServiceInstanceIDKey),
	} {
		if value, ok := defaults[name]; ok {
			resourceCfg.Attributes = append(resourceCfg.Attributes, config.AttributeNameValue{
				Name:  name,
				Value: value,
			})
		}
	}

	legacyKeys := make([]string, 0, len(cfg.LegacyAttributes))
	for key := range cfg.LegacyAttributes {
		legacyKeys = append(legacyKeys, key)
	}
	sort.Strings(legacyKeys)
	for _, key := range legacyKeys {
		value := cfg.LegacyAttributes[key]
		if value == nil {
			continue
		}
		resourceCfg.Attributes = append(resourceCfg.Attributes, config.AttributeNameValue{
			Name:  key,
			Value: value,
		})
	}

	resourceCfg.Attributes = append(resourceCfg.Attributes, cfg.Attributes...)
	return resourceCfg, nil
}

func resourceConfigFromSettings(set telemetry.Settings, cfg *Config) config.Resource {
	resourceCfg := config.Resource{
		SchemaUrl: cfg.Resource.SchemaUrl,
	}
	if resourceCfg.SchemaUrl == nil {
		resourceCfg.SchemaUrl = ptr(semconv.SchemaURL)
	}
	if set.Resource == nil {
		return resourceCfg
	}

	set.Resource.Attributes().Range(func(k string, v pcommon.Value) bool {
		resourceCfg.Attributes = append(resourceCfg.Attributes, config.AttributeNameValue{
			Name:  k,
			Value: v.AsRaw(),
		})
		return true
	})
	return resourceCfg
}
