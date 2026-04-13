// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"

	otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	internalresource "go.opentelemetry.io/collector/service/internal/resource"
	"go.opentelemetry.io/collector/service/telemetry"
)

var defaultAttributeValues = internalresource.DefaultAttributeValues

func buildSDKResourceConfig(buildInfo component.BuildInfo, cfg *ResourceConfig) (*otelconf.Resource, error) {
	defaults, err := defaultAttributeValues(buildInfo)
	if err != nil {
		return nil, err
	}

	sdkCfg := cfg.Resource
	maxAttributes := len(cfg.Attributes) + len(cfg.LegacyAttributes) + len(defaults)
	sdkCfg.Attributes = make([]otelconf.AttributeNameValue, 0, maxAttributes)

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
	return &sdkCfg, nil
}

func createResourceConfig(
	ctx context.Context,
	buildInfo component.BuildInfo,
	cfg *ResourceConfig,
	res *pcommon.Resource,
) (*otelconf.Resource, error) {
	if res == nil {
		generated, err := createConfiguredResource(ctx, buildInfo, cfg)
		if err != nil {
			return nil, err
		}
		res = &generated
	}

	providerConfig := &otelconf.Resource{
		Attributes: make([]otelconf.AttributeNameValue, 0, res.Attributes().Len()),
	}
	if cfg.SchemaUrl != nil {
		schemaURL := *cfg.SchemaUrl
		providerConfig.SchemaUrl = &schemaURL
	}

	res.Attributes().Range(func(key string, value pcommon.Value) bool {
		providerConfig.Attributes = append(providerConfig.Attributes, otelconf.AttributeNameValue{
			Name:  key,
			Value: value.AsRaw(),
		})
		return true
	})

	return providerConfig, nil
}

func createConfiguredResource(
	ctx context.Context,
	buildInfo component.BuildInfo,
	cfg *ResourceConfig,
) (pcommon.Resource, error) {
	sdkCfg, err := buildSDKResourceConfig(buildInfo, cfg)
	if err != nil {
		return pcommon.Resource{}, err
	}

	sdk, err := otelconf.NewSDK(otelconf.WithContext(ctx), otelconf.WithOpenTelemetryConfiguration(otelconf.OpenTelemetryConfiguration{
		Resource: sdkCfg,
	}))
	if err != nil {
		return pcommon.Resource{}, err
	}

	sdkResource := sdk.Resource()
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

func (f *otelconfFactory) createResource(
	ctx context.Context,
	set telemetry.Settings,
	componentConfig component.Config,
) (pcommon.Resource, error) {
	return createConfiguredResource(ctx, set.BuildInfo, &componentConfig.(*Config).Resource)
}
