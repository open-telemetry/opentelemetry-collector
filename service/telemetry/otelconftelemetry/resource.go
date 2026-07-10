// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"
	"errors"

	"github.com/google/uuid"
	otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	xotelconf "go.opentelemetry.io/contrib/otelconf/x"
	"go.opentelemetry.io/otel/attribute"
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

func createInitialResourceConfig(buildInfo component.BuildInfo, cfg *ResourceConfig) (*otelconf.Resource, error) {
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

func createResource(
	ctx context.Context,
	set telemetry.Settings,
	componentConfig component.Config,
) (pcommon.Resource, error) {
	cfg := &componentConfig.(*Config).Resource
	sdkCfg, err := createInitialResourceConfig(set.BuildInfo, cfg)
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
	if cfg.DetectionDevelopment != nil {
		detectionSDK, err := newDetectionResourceSDK(ctx, cfg.DetectionDevelopment)
		if err != nil {
			return pcommon.Resource{}, err
		}
		detectedResource := otelsdkresource.NewSchemaless(detectionSDK.Resource().Attributes()...)
		sdkResource, err = otelsdkresource.Merge(detectedResource, sdkResource)
		if err != nil {
			return pcommon.Resource{}, err
		}
	}

	sdkIterator := sdkResource.Iter()

	pcommonResource := pcommon.NewResource()
	pcommonAttributes := pcommonResource.Attributes()
	pcommonAttributes.EnsureCapacity(sdkIterator.Len())

	for sdkIterator.Next() {
		kv := sdkIterator.Attribute()
		putSDKAttribute(pcommonAttributes, string(kv.Key), kv.Value)
	}

	return pcommonResource, nil
}

// putSDKAttribute copies an OTel SDK attribute value into a pcommon.Map, handling every
// attribute type explicitly (mirroring internal/telemetry.ToZapFields). Slice-typed values
// (as emitted by resource detectors such as the process detector's process.command_args) are
// built element-wise because pcommon.Value.FromRaw only accepts []any, not the typed scalar
// slices returned by attribute.Value.AsInterface().
func putSDKAttribute(attrs pcommon.Map, key string, value attribute.Value) {
	switch value.Type() {
	case attribute.BOOL:
		attrs.PutBool(key, value.AsBool())
	case attribute.INT64:
		attrs.PutInt(key, value.AsInt64())
	case attribute.FLOAT64:
		attrs.PutDouble(key, value.AsFloat64())
	case attribute.STRING:
		attrs.PutStr(key, value.AsString())
	case attribute.BOOLSLICE:
		slice := attrs.PutEmptySlice(key)
		for _, v := range value.AsBoolSlice() {
			slice.AppendEmpty().SetBool(v)
		}
	case attribute.INT64SLICE:
		slice := attrs.PutEmptySlice(key)
		for _, v := range value.AsInt64Slice() {
			slice.AppendEmpty().SetInt(v)
		}
	case attribute.FLOAT64SLICE:
		slice := attrs.PutEmptySlice(key)
		for _, v := range value.AsFloat64Slice() {
			slice.AppendEmpty().SetDouble(v)
		}
	case attribute.STRINGSLICE:
		slice := attrs.PutEmptySlice(key)
		for _, v := range value.AsStringSlice() {
			slice.AppendEmpty().SetStr(v)
		}
	default:
		attrs.PutEmpty(key)
	}
}

func newDetectionResourceSDK(ctx context.Context, detection *xotelconf.ExperimentalResourceDetection) (xotelconf.SDK, error) {
	return newExperimentalSDK(
		xotelconf.WithContext(ctx),
		xotelconf.WithOpenTelemetryConfiguration(xotelconf.OpenTelemetryConfiguration{
			Resource: &xotelconf.Resource{DetectionDevelopment: detection},
		}),
	)
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
