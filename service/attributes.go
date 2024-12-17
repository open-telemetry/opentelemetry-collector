// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service // import "go.opentelemetry.io/collector/service"

import (
	sdkresource "go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/collector/service/telemetry"
)

func attributes(res *sdkresource.Resource, cfg telemetry.Config) map[string]any {
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
	return attrs
}
