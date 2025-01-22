// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service // import "go.opentelemetry.io/collector/service"

import (
	config "go.opentelemetry.io/contrib/config/v0.3.0"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/collector/service/telemetry"
)

func attributes(res *sdkresource.Resource, cfg telemetry.Config) []config.AttributeNameValue {
	attrs := map[string]any{}
	for _, r := range res.Attributes() {
		attrs[string(r.Key)] = r.Value.AsString()
	}

	for k, v := range cfg.Resource {
		if v != nil {
			attrs[k] = *v
		} else {
			// the new value is nil, delete the existing key
			delete(attrs, k)
		}
	}

	attrSlice := make([]config.AttributeNameValue, 0, len(res.Attributes())+len(cfg.Resource))

	for k, v := range attrs {
		attrSlice = append(attrSlice, config.AttributeNameValue{Name: k, Value: v})
	}

	return attrSlice
}
