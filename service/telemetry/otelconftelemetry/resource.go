// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	xotelconf "go.opentelemetry.io/contrib/otelconf/x"
	otelsdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
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

var newExperimentalSDK = xotelconf.NewSDK

var putAttributeValue = func(attrs pcommon.Map, key string, value any) error {
	return attrs.PutEmpty(key).FromRaw(value)
}

func createInitialResourceConfig(_ context.Context, _ component.BuildInfo, cfg *ResourceConfig) (*xotelconf.Resource, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &xotelconf.Resource{
		DetectionDevelopment: cfg.DetectionDevelopment,
	}, nil
}

func createFixedResourceConfig(cfg *ResourceConfig, res *pcommon.Resource) (*otelconf.Resource, error) {
	if res == nil {
		return nil, errMissingCollectorResource
	}

	providerConfig := &otelconf.Resource{
		Attributes: make([]otelconf.AttributeNameValue, 0, res.Attributes().Len()),
	}

	res.Attributes().Range(func(key string, value pcommon.Value) bool {
		providerConfig.Attributes = append(providerConfig.Attributes, otelconf.AttributeNameValue{
			Name:  key,
			Value: value.AsRaw(),
		})
		return true
	})
	if cfg.SchemaUrl != nil {
		// Preserve the configured schema URL separately because it is not exposed by pcommon.Resource.
		schemaURL := *cfg.SchemaUrl
		providerConfig.SchemaUrl = &schemaURL
	}

	return providerConfig, nil
}

func createResource(
	ctx context.Context,
	set telemetry.Settings,
	componentConfig component.Config,
) (pcommon.Resource, error) {
	cfg := &componentConfig.(*Config).Resource
	sdkCfg, err := createInitialResourceConfig(ctx, set.BuildInfo, &componentConfig.(*Config).Resource)
	if err != nil {
		return pcommon.Resource{}, err
	}
	sdk, err := newResourceSDK(ctx, sdkCfg)
	if err != nil {
		return pcommon.Resource{}, err
	}

	defaults, err := defaultAttributeValues(set.BuildInfo)
	if err != nil {
		return pcommon.Resource{}, err
	}
	envResource := otelsdkresource.Environment()
	suppressed := suppressedLegacyAttributes(cfg.LegacyAttributes)

	sdkResource := sdk.Resource()
	pcommonResource := pcommon.NewResource()
	pcommonAttributes := pcommonResource.Attributes()

	pcommonAttributes.EnsureCapacity(len(defaults) + len(cfg.LegacyAttributes) + len(cfg.Attributes) + sdkResource.Len() + envResource.Len())

	for name, value := range defaults {
		if _, ok := suppressed[name]; ok {
			continue
		}
		if err := putAttributeValue(pcommonAttributes, name, normalizeAttributeValue(value)); err != nil {
			return pcommon.Resource{}, err
		}
	}

	envIterator := envResource.Iter()
	for envIterator.Next() {
		kv := envIterator.Attribute()
		if _, ok := suppressed[string(kv.Key)]; ok {
			continue
		}
		if err := putAttributeValue(pcommonAttributes, string(kv.Key), kv.Value.AsInterface()); err != nil {
			return pcommon.Resource{}, err
		}
	}

	sdkIterator := sdkResource.Iter()

	for sdkIterator.Next() {
		kv := sdkIterator.Attribute()
		key := string(kv.Key)
		if _, ok := suppressed[key]; ok {
			continue
		}
		if _, exists := pcommonAttributes.Get(key); exists {
			continue
		}
		if err := putAttributeValue(pcommonAttributes, key, kv.Value.AsInterface()); err != nil {
			return pcommon.Resource{}, err
		}
	}

	for key, value := range cfg.LegacyAttributes {
		if value == nil {
			continue
		}
		if err := putAttributeValue(pcommonAttributes, key, normalizeAttributeValue(value)); err != nil {
			return pcommon.Resource{}, err
		}
	}

	for _, attr := range cfg.Attributes {
		if err := putAttributeValue(pcommonAttributes, attr.Name, normalizeAttributeValue(attr.Value)); err != nil {
			return pcommon.Resource{}, err
		}
	}

	return pcommonResource, nil
}

func newResourceSDK(ctx context.Context, resourceCfg *xotelconf.Resource) (xotelconf.SDK, error) {
	return newExperimentalSDK(
		xotelconf.WithContext(ctx),
		xotelconf.WithOpenTelemetryConfiguration(xotelconf.OpenTelemetryConfiguration{Resource: resourceCfg}),
	)
}

func suppressedLegacyAttributes(legacy map[string]any) map[string]struct{} {
	suppressed := make(map[string]struct{}, len(legacy))
	for name := range legacy {
		suppressed[name] = struct{}{}
	}
	return suppressed
}

func normalizeAttributeValue(v any) any {
	switch val := v.(type) {
	case bool, int64, float64, string:
		return val
	case uint64:
		return strconv.FormatUint(val, 10)
	case int8:
		return int64(val)
	case uint8:
		return int64(val)
	case int16:
		return int64(val)
	case uint16:
		return int64(val)
	case int32:
		return int64(val)
	case uint32:
		return int64(val)
	case float32:
		return float64(val)
	case int:
		return int64(val)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	default:
		return fmt.Sprint(v)
	}
}
