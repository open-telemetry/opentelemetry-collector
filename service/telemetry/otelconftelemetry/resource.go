// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"
	"sync"

	otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	internalresource "go.opentelemetry.io/collector/service/internal/resource"
	"go.opentelemetry.io/collector/service/telemetry"
)

var defaultAttributeValues = internalresource.DefaultAttributeValues

type fullResource struct {
	sdkResource    *resource.Resource
	providerConfig *otelconf.Resource
}

type resourceCache struct {
	once         sync.Once
	fullResource *fullResource
	err          error
}

func createFullResource(
	ctx context.Context,
	buildInfo component.BuildInfo,
	cfg *ResourceConfig,
) (*fullResource, error) {
	// Compute default values
	defaults, err := defaultAttributeValues(buildInfo)
	if err != nil {
		return nil, err
	}

	// Generate final SDK resource config, taking into account default values and legacy (map-style) attributes.
	// To do this, we start with a shallow copy, and rebuild the attributes list.
	sdkCfg := cfg.Resource
	maxAttributes := len(cfg.Attributes) + len(cfg.LegacyAttributes) + len(defaults)
	sdkCfg.Attributes = make([]otelconf.AttributeNameValue, 0, maxAttributes)

	// Attribute order matters: later entries overwrite earlier ones in the SDK.
	// We rely on this behavior to prioritize declarative attributes over legacy ones,
	// and legacy ones over defaults.

	for name, value := range defaults {
		if _, ok := cfg.LegacyAttributes[name]; ok {
			continue
		}
		sdkCfg.Attributes = append(sdkCfg.Attributes, otelconf.AttributeNameValue{
			Name:  name,
			Value: value,
		})
	}

	for key, value := range cfg.LegacyAttributes {
		if value == nil {
			continue
		}
		sdkCfg.Attributes = append(sdkCfg.Attributes, otelconf.AttributeNameValue{
			Name:  key,
			Value: value,
		})
	}

	sdkCfg.Attributes = append(sdkCfg.Attributes, cfg.Attributes...)

	// Generate final SDK resource using otelconf
	sdk, err := otelconf.NewSDK(otelconf.WithContext(ctx), otelconf.WithOpenTelemetryConfiguration(otelconf.OpenTelemetryConfiguration{
		Resource: &sdkCfg,
	}))
	if err != nil {
		return nil, err
	}

	// Convert SDK resource back into an equivalent SDK config for provider instantiation.
	sdkResource := sdk.Resource()
	sdkIterator := sdkResource.Iter()
	providerConfig := &otelconf.Resource{
		Attributes: make([]otelconf.AttributeNameValue, 0, sdkIterator.Len()),
	}
	if schemaURL := sdkResource.SchemaURL(); schemaURL != "" {
		providerConfig.SchemaUrl = &schemaURL
	}
	for sdkIterator.Next() {
		kv := sdkIterator.Attribute()
		providerConfig.Attributes = append(providerConfig.Attributes, otelconf.AttributeNameValue{
			Name:  string(kv.Key),
			Value: kv.Value.AsInterface(),
		})
	}

	return &fullResource{
		sdkResource:    sdkResource,
		providerConfig: providerConfig,
	}, nil
}

func (f *otelconfFactory) createResourceConfigOnce(
	ctx context.Context,
	buildInfo component.BuildInfo,
	componentConfig component.Config,
) (*otelconf.Resource, error) {
	f.resourceCache.once.Do(func() {
		f.resourceCache.fullResource, f.resourceCache.err = createFullResource(ctx, buildInfo, &componentConfig.(*Config).Resource)
	})
	return f.resourceCache.fullResource.providerConfig, f.resourceCache.err
}

func (f *otelconfFactory) createResource(
	ctx context.Context,
	set telemetry.Settings,
	componentConfig component.Config,
) (pcommon.Resource, error) {
	_, err := f.createResourceConfigOnce(ctx, set.BuildInfo, componentConfig)
	if err != nil {
		return pcommon.Resource{}, err
	}

	sdkResource := f.resourceCache.fullResource.sdkResource
	sdkIterator := sdkResource.Iter()

	pcommonResource := pcommon.NewResource()
	pcommonAttributes := pcommonResource.Attributes()
	pcommonAttributes.EnsureCapacity(sdkIterator.Len())

	for sdkIterator.Next() {
		kv := sdkIterator.Attribute()
		if err := pcommonAttributes.PutEmpty(string(kv.Key)).FromRaw(kv.Value.AsInterface()); err != nil {
			return pcommon.Resource{}, err
		}
	}

	return pcommonResource, nil
}
