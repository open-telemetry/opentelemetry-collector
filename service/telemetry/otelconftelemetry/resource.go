// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"
	"errors"

	"github.com/google/uuid"
	otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	collectorresource "go.opentelemetry.io/collector/service/internal/resource"
	"go.opentelemetry.io/collector/service/telemetry"
)

var errMissingCollectorResource = errors.New("collector resource must be initialized before creating telemetry providers")

// defaultAttributeValues is a variable so tests can stub the generated defaults.
var defaultAttributeValues = func(buildInfo component.BuildInfo) (map[string]string, error) {
	instanceUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	return map[string]string{
		string(semconv.ServiceNameKey):       buildInfo.Command,
		string(semconv.ServiceVersionKey):    buildInfo.Version,
		string(semconv.ServiceInstanceIDKey): instanceUUID.String(),
	}, nil
}

func createInitialResourceConfig(ctx context.Context, buildInfo component.BuildInfo, cfg *ResourceConfig) (*otelconf.Resource, error) {
	defaults, err := defaultAttributeValues(buildInfo)
	if err != nil {
		return nil, err
	}
	detectedAttributes, err := detectorAttributeValues(ctx, cfg)
	if err != nil {
		return nil, err
	}

	sdkCfg := cfg.Resource
	maxAttributes := len(cfg.Attributes) + len(cfg.LegacyAttributes) + len(defaults) + len(detectedAttributes)
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

	sdkCfg.Attributes = append(sdkCfg.Attributes, detectedAttributes...)

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

func detectorAttributeValues(ctx context.Context, cfg *ResourceConfig) ([]otelconf.AttributeNameValue, error) {
	detectorNames := configuredDetectorNames(cfg.Detectors)
	if len(detectorNames) == 0 {
		return nil, nil
	}

	detectors, err := collectorresource.GetDetectors(ctx, detectorNames)
	if err != nil {
		return nil, err
	}

	suppressed := make(map[string]struct{}, len(cfg.LegacyAttributes))
	for name := range cfg.LegacyAttributes {
		suppressed[name] = struct{}{}
	}

	attrs := make([]otelconf.AttributeNameValue, 0, len(detectors))
	for _, detector := range detectors {
		res, err := detector.Detect(ctx)
		if err != nil {
			if res == nil || !errors.Is(err, sdkresource.ErrPartialResource) {
				return nil, err
			}
		}
		attrs = append(attrs, detectedResourceToConfig(res, suppressed)...)
	}

	return attrs, nil
}

func configuredDetectorNames(detectors *otelconf.Detectors) []string {
	if detectors == nil || detectors.Attributes == nil || len(detectors.Attributes.Included) == 0 {
		return nil
	}

	excluded := make(map[string]struct{}, len(detectors.Attributes.Excluded))
	for _, name := range detectors.Attributes.Excluded {
		excluded[name] = struct{}{}
	}

	names := make([]string, 0, len(detectors.Attributes.Included))
	for _, name := range detectors.Attributes.Included {
		if _, ok := excluded[name]; ok {
			continue
		}
		names = append(names, name)
	}
	return names
}

func detectedResourceToConfig(res *sdkresource.Resource, suppressed map[string]struct{}) []otelconf.AttributeNameValue {
	if res == nil {
		return nil
	}

	attrs := make([]otelconf.AttributeNameValue, 0, len(res.Attributes()))
	for _, attr := range res.Attributes() {
		name := string(attr.Key)
		if _, ok := suppressed[name]; ok {
			continue
		}
		attrs = append(attrs, otelconf.AttributeNameValue{
			Name:  name,
			Value: attr.Value.AsInterface(),
		})
	}
	return attrs
}

func createFixedResourceConfig(cfg *ResourceConfig, res *pcommon.Resource) (*otelconf.Resource, error) {
	if res == nil {
		return nil, errMissingCollectorResource
	}

	providerConfig := &otelconf.Resource{
		Attributes: make([]otelconf.AttributeNameValue, 0, res.Attributes().Len()),
	}
	if cfg.SchemaUrl != nil {
		// Preserve the configured schema URL separately because it is not exposed by pcommon.Resource.
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

func createResource(
	ctx context.Context,
	set telemetry.Settings,
	componentConfig component.Config,
) (pcommon.Resource, error) {
	sdkCfg, err := createInitialResourceConfig(ctx, set.BuildInfo, &componentConfig.(*Config).Resource)
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
