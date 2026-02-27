// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"

	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
)

func newSDK(ctx context.Context, resCfg *config.Resource, conf config.OpenTelemetryConfiguration) (config.SDK, error) {
	if resCfg != nil {
		conf.Resource = resCfg
	}
	return config.NewSDK(config.WithContext(ctx), config.WithOpenTelemetryConfiguration(conf))
}
