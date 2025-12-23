// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"
	"fmt"

	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel/attribute"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
)

func newSDK(ctx context.Context, res *sdkresource.Resource, conf config.OpenTelemetryConfiguration) (config.SDK, error) {
	resourceAttrs := make([]config.AttributeNameValue, 0, res.Len())
	for _, r := range res.Attributes() {
		attr, err := attributeToConfig(r)
		if err != nil {
			return config.SDK{}, err
		}
		resourceAttrs = append(resourceAttrs, attr)
	}
	conf.Resource = &config.Resource{
		SchemaUrl:  ptr(semconv.SchemaURL),
		Attributes: resourceAttrs,
	}
	return config.NewSDK(config.WithContext(ctx), config.WithOpenTelemetryConfiguration(conf))
}

func attributeToConfig(kv attribute.KeyValue) (config.AttributeNameValue, error) {
	result := config.AttributeNameValue{Name: string(kv.Key)}
	switch kv.Value.Type() {
	case attribute.STRING:
		result.Value = kv.Value.AsString()
	case attribute.BOOL:
		result.Value = kv.Value.AsBool()
		result.Type = &config.AttributeNameValueType{Value: "bool"}
	case attribute.INT64:
		result.Value = kv.Value.AsInt64()
		result.Type = &config.AttributeNameValueType{Value: "int"}
	case attribute.FLOAT64:
		result.Value = kv.Value.AsFloat64()
		result.Type = &config.AttributeNameValueType{Value: "double"}
	case attribute.STRINGSLICE:
		result.Value = sliceToAny(kv.Value.AsStringSlice())
		result.Type = &config.AttributeNameValueType{Value: "string_array"}
	case attribute.BOOLSLICE:
		result.Value = sliceToAny(kv.Value.AsBoolSlice())
		result.Type = &config.AttributeNameValueType{Value: "bool_array"}
	case attribute.INT64SLICE:
		result.Value = sliceToAny(kv.Value.AsInt64Slice())
		result.Type = &config.AttributeNameValueType{Value: "int_array"}
	case attribute.FLOAT64SLICE:
		result.Value = sliceToAny(kv.Value.AsFloat64Slice())
		result.Type = &config.AttributeNameValueType{Value: "double_array"}
	default:
		return config.AttributeNameValue{}, fmt.Errorf("unsupported attribute type %s for %q", kv.Value.Type(), kv.Key)
	}
	return result, nil
}

func sliceToAny[T any](vals []T) []any {
	out := make([]any, len(vals))
	for i, v := range vals {
		out[i] = v
	}
	return out
}
