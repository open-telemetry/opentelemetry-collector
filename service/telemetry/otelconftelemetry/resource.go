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
	_ context.Context,
	set telemetry.Settings,
	componentConfig component.Config,
) (pcommon.Resource, error) {
	res := newResource(set, componentConfig.(*Config))
	pcommonRes := pcommon.NewResource()
	for _, keyValue := range res.Attributes() {
		key := string(keyValue.Key)
		value, err := attributeValueString(key, keyValue.Value)
		if err != nil {
			return pcommon.Resource{}, err
		}
		pcommonRes.Attributes().PutStr(key, value)
	}
	return pcommonRes, nil
}

func newResource(set telemetry.Settings, cfg *Config) *sdkresource.Resource {
	return resource.New(set.BuildInfo, cfg.Resource)
}

func attributeValueString(k string, v attribute.Value) (string, error) {
	if v.Type() != attribute.STRING {
		// We only support string-type resource attributes in the configuration.
		return "", fmt.Errorf("attribute %q: expected string, got %s", k, v.Type())
	}
	return v.AsString(), nil
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
