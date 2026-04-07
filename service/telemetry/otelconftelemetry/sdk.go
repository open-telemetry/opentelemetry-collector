// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"

	otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"
)

func newSDK(ctx context.Context, resourceConfig *otelconf.Resource, conf otelconf.OpenTelemetryConfiguration) (otelconf.SDK, error) {
	conf.Resource = resourceConfig
	return otelconf.NewSDK(otelconf.WithContext(ctx), otelconf.WithOpenTelemetryConfiguration(conf))
}
