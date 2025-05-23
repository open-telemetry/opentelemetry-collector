// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resource // import "go.opentelemetry.io/collector/service/internal/resource"

import (
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"

	"go.opentelemetry.io/collector/component"
)

// New resource from telemetry configuration.
func New(buildInfo component.BuildInfo, resourceCfg map[string]*string) *resource.Resource {
	var telAttrs []attribute.KeyValue

	for k, v := range resourceCfg {
		// nil value indicates that the attribute should not be included in the telemetry.
		if v != nil {
			telAttrs = append(telAttrs, attribute.String(k, *v))
		}
	}

	if _, ok := resourceCfg[string(semconv.ServiceNameKey)]; !ok {
		// AttributeServiceName is not specified in the config. Use the default service name.
		telAttrs = append(telAttrs, semconv.ServiceNameKey.String(buildInfo.Command))
	}

	if _, ok := resourceCfg[string(semconv.ServiceInstanceIDKey)]; !ok {
		// AttributeServiceInstanceID is not specified in the config. Auto-generate one.
		instanceUUID, _ := uuid.NewRandom()
		instanceID := instanceUUID.String()
		telAttrs = append(telAttrs, semconv.ServiceInstanceIDKey.String(instanceID))
	}

	if _, ok := resourceCfg[string(semconv.ServiceVersionKey)]; !ok {
		// AttributeServiceVersion is not specified in the config. Use the actual
		// build version.
		telAttrs = append(telAttrs, semconv.ServiceVersionKey.String(buildInfo.Version))
	}
	return resource.NewWithAttributes(semconv.SchemaURL, telAttrs...)
}
