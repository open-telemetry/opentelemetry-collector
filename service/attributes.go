// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service // import "go.opentelemetry.io/collector/service"

import (
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/collector/service/telemetry"
)

func attributes(res *sdkresource.Resource, cfg telemetry.Config) []config.AttributeNameValue {
	attrsMap := map[string]any{}
	for _, r := range res.Attributes() {
		attrsMap[string(r.Key)] = r.Value.AsString()
	}

	for k, v := range cfg.Resource {
		if v != nil {
			attrsMap[k] = *v
		} else {
			// the new value is nil, delete the existing key
			delete(attrsMap, k)
		}
	}

	var attrs []config.AttributeNameValue
	for k, v := range attrsMap {
		attrs = append(attrs, config.AttributeNameValue{Name: k, Value: v})
	}

	return attrs
}
