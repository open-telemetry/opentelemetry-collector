// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

func attributes(set Settings, cfg Config) map[string]interface{} {
	attrs := map[string]interface{}{
		string(semconv.ServiceNameKey):    set.BuildInfo.Command,
		string(semconv.ServiceVersionKey): set.BuildInfo.Version,
	}
	for k, v := range cfg.Resource {
		if v != nil {
			attrs[k] = *v
		}

		// the new value is nil, delete the existing key
		if _, ok := attrs[k]; ok && v == nil {
			delete(attrs, k)
		}
	}
	return attrs
}
