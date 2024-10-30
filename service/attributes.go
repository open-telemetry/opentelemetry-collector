// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service // import "go.opentelemetry.io/collector/service"

import (
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/telemetry"
)

func attributes(buildInfo component.BuildInfo, cfg telemetry.Config) map[string]interface{} {
	attrs := map[string]interface{}{
		string(semconv.ServiceNameKey):    buildInfo.Command,
		string(semconv.ServiceVersionKey): buildInfo.Version,
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
