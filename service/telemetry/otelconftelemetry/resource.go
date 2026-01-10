// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/service/internal/resource"
	"go.opentelemetry.io/collector/service/telemetry"
)

func createResource(
	ctx context.Context,
	set telemetry.Settings,
	componentConfig component.Config,
) (pcommon.Resource, error) {
	res, err := newResource(ctx, set, componentConfig.(*Config))
	if err != nil {
		return pcommon.Resource{}, fmt.Errorf("failed to create resource: %w", err)
	}
	pcommonRes := pcommon.NewResource()
	for _, keyValue := range res.Attributes() {
		setAttributeValue(pcommonRes.Attributes(), string(keyValue.Key), keyValue.Value)
	}
	return pcommonRes, nil
}

func newResource(ctx context.Context, set telemetry.Settings, cfg *Config) (*sdkresource.Resource, error) {
	return resource.New(ctx, set.BuildInfo, &cfg.Resource)
}

func setAttributeValue(attrs pcommon.Map, key string, value attribute.Value) {
	switch value.Type() {
	case attribute.STRING:
		attrs.PutStr(key, value.AsString())
	case attribute.BOOL:
		attrs.PutBool(key, value.AsBool())
	case attribute.INT64:
		attrs.PutInt(key, value.AsInt64())
	case attribute.FLOAT64:
		attrs.PutDouble(key, value.AsFloat64())
	case attribute.STRINGSLICE:
		slice := attrs.PutEmptySlice(key)
		for _, v := range value.AsStringSlice() {
			slice.AppendEmpty().SetStr(v)
		}
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
	default:
		panic(fmt.Errorf("attribute %q: unsupported attribute type %s", key, value.Type()))
	}
}

// pcommonAttrsToOTelAttrs gets the Resource attributes to OpenTelemetry attribute.KeyValue slice.
func pcommonAttrsToOTelAttrs(resource *pcommon.Resource) []attribute.KeyValue {
	var result []attribute.KeyValue
	if resource != nil {
		attrs := resource.Attributes()
		attrs.Range(func(k string, v pcommon.Value) bool {
			result = append(result, attribute.String(k, v.AsString()))
			return true
		})
	}
	return result
}
