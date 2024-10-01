// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"net/http"
	"sync"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/service/telemetry/internal/otelinit"
)

const (
	zapKeyTelemetryAddress = "address"
	zapKeyTelemetryLevel   = "metrics level"
)

type meterProvider struct {
	*sdkmetric.MeterProvider
	servers  []*http.Server
	serverWG sync.WaitGroup
}

type meterProviderSettings struct {
	res               *resource.Resource
	cfg               MetricsConfig
	asyncErrorChannel chan error
}

func newMeterProvider(set meterProviderSettings, disableHighCardinality bool) (metric.MeterProvider, error) {
	if set.cfg.Level == configtelemetry.LevelNone || len(set.cfg.Readers) == 0 {
		return noop.NewMeterProvider(), nil
	}

	mp := &meterProvider{}
	var opts []sdkmetric.Option
	for _, reader := range set.cfg.Readers {
		// https://github.com/open-telemetry/opentelemetry-collector/issues/8045
		r, server, err := otelinit.InitMetricReader(context.Background(), reader, set.asyncErrorChannel, &mp.serverWG)
		if err != nil {
			return nil, err
		}
		if server != nil {
			mp.servers = append(mp.servers, server)

		}
		opts = append(opts, sdkmetric.WithReader(r))
	}

	var err error
	mp.MeterProvider, err = otelinit.InitOpenTelemetry(set.res, opts, disableHighCardinality)
	if err != nil {
		return nil, err
	}
	return mp, nil
}

// LogAboutServers logs about the servers that are serving metrics.
func (mp *meterProvider) LogAboutServers(logger *zap.Logger, cfg MetricsConfig) {
	for _, server := range mp.servers {
		logger.Info(
			"Serving metrics",
			zap.String(zapKeyTelemetryAddress, server.Addr),
			zap.Stringer(zapKeyTelemetryLevel, cfg.Level),
		)
	}
}

// Shutdown the meter provider and all the associated resources.
// The type signature of this method matches that of the sdkmetric.MeterProvider.
func (mp *meterProvider) Shutdown(ctx context.Context) error {
	var errs error
	for _, server := range mp.servers {
		if server != nil {
			errs = multierr.Append(errs, server.Close())
		}
	}
	errs = multierr.Append(errs, mp.MeterProvider.Shutdown(ctx))
	mp.serverWG.Wait()

	return errs
}
