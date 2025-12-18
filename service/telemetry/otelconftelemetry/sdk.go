// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"

	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
)

func newSDK(ctx context.Context, res *sdkresource.Resource, conf config.OpenTelemetryConfiguration) (config.SDK, error) {
	resourceAttrs := make([]config.AttributeNameValue, 0, res.Len())
	for _, r := range res.Attributes() {
		key := string(r.Key)
		resourceAttrs = append(resourceAttrs, config.AttributeNameValue{
			Name:  key,
			Value: mustAttributeValueString(key, r.Value),
		})
	}
	conf.Resource = &config.Resource{
		SchemaUrl:  ptr(semconv.SchemaURL),
		Attributes: resourceAttrs,
	}
	return config.NewSDK(config.WithContext(ctx), config.WithOpenTelemetryConfiguration(conf))
}
