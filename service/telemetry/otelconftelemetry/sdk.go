// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"

	"go.opentelemetry.io/contrib/otelconf"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func newSDK(ctx context.Context, res *sdkresource.Resource, conf otelconf.OpenTelemetryConfiguration) (otelconf.SDK, error) {
	resourceAttrs := make([]otelconf.AttributeNameValue, 0, res.Len())
	for _, r := range res.Attributes() {
		key := string(r.Key)
		resourceAttrs = append(resourceAttrs, otelconf.AttributeNameValue{
			Name:  key,
			Value: mustAttributeValueString(key, r.Value),
		})
	}
	conf.Resource = &otelconf.ResourceJson{
		SchemaUrl:  ptr(semconv.SchemaURL),
		Attributes: resourceAttrs,
	}
	return otelconf.NewSDK(otelconf.WithContext(ctx), otelconf.WithOpenTelemetryConfiguration(conf))
}
