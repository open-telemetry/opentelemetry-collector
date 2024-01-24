// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resource // import "go.opentelemetry.io/collector/service/internal/resource"

import (
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/collector/component"
	semconv "go.opentelemetry.io/collector/semconv/v1.18.0"
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

	if _, ok := resourceCfg[semconv.AttributeServiceName]; !ok {
		// AttributeServiceName is not specified in the config. Use the default service name.
		telAttrs = append(telAttrs, attribute.String(semconv.AttributeServiceName, buildInfo.Command))
	}

	if _, ok := resourceCfg[semconv.AttributeServiceInstanceID]; !ok {
		// AttributeServiceInstanceID is not specified in the config. Auto-generate one.
		instanceUUID, _ := uuid.NewRandom()
		instanceID := instanceUUID.String()
		telAttrs = append(telAttrs, attribute.String(semconv.AttributeServiceInstanceID, instanceID))
	}

	if _, ok := resourceCfg[semconv.AttributeServiceVersion]; !ok {
		// AttributeServiceVersion is not specified in the config. Use the actual
		// build version.
		telAttrs = append(telAttrs, attribute.String(semconv.AttributeServiceVersion, buildInfo.Version))
	}
	return resource.NewWithAttributes(semconv.SchemaURL, telAttrs...)
}
