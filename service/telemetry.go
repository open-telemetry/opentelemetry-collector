// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service // import "go.opentelemetry.io/collector/service"

import (
	"context"
	"net"
	"net/http"
	"strconv"

	ocmetric "go.opencensus.io/metric"
	"go.opencensus.io/metric/metricproducer"
	"go.opentelemetry.io/contrib/config"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/service/internal/proctelemetry"
	"go.opentelemetry.io/collector/service/telemetry"
)

const (
	zapKeyTelemetryAddress = "address"
	zapKeyTelemetryLevel   = "level"
)

type meterProvider struct {
	*sdkmetric.MeterProvider
	ocRegistry *ocmetric.Registry
	servers    []*http.Server
}

type meterProviderSettings struct {
	res               *resource.Resource
	logger            *zap.Logger
	cfg               telemetry.MetricsConfig
	asyncErrorChannel chan error
}

func newMeterProvider(set meterProviderSettings, disableHighCardinality bool, extendedConfig bool) (metric.MeterProvider, error) {
	if set.cfg.Level == configtelemetry.LevelNone || (set.cfg.Address == "" && len(set.cfg.Readers) == 0) {
		set.logger.Info(
			"Skipping telemetry setup.",
			zap.String(zapKeyTelemetryAddress, set.cfg.Address),
			zap.String(zapKeyTelemetryLevel, set.cfg.Level.String()),
		)
		return noopmetric.NewMeterProvider(), nil
	}

	set.logger.Info("Setting up own telemetry...")
	if len(set.cfg.Address) != 0 {
		if extendedConfig {
			set.logger.Warn("service::telemetry::metrics::address is being deprecated in favor of service::telemetry::metrics::readers")
		}
		host, port, err := net.SplitHostPort(set.cfg.Address)
		if err != nil {
			return nil, err
		}
		portInt, err := strconv.Atoi(port)
		if err != nil {
			return nil, err
		}
		if set.cfg.Readers == nil {
			set.cfg.Readers = []config.MetricReader{}
		}
		set.cfg.Readers = append(set.cfg.Readers, config.MetricReader{
			Pull: &config.PullMetricReader{
				Exporter: config.MetricExporter{
					Prometheus: &config.Prometheus{
						Host: &host,
						Port: &portInt,
					},
				},
			},
		})
	}

	mp := &meterProvider{
		// Initialize the ocRegistry, still used by the process metrics.
		ocRegistry: ocmetric.NewRegistry(),
	}
	metricproducer.GlobalManager().AddProducer(mp.ocRegistry)
	opts := []sdkmetric.Option{}
	for _, reader := range set.cfg.Readers {
		// https://github.com/open-telemetry/opentelemetry-collector/issues/8045
		r, server, err := proctelemetry.InitMetricReader(context.Background(), reader, set.asyncErrorChannel)
		if err != nil {
			return nil, err
		}
		if server != nil {
			mp.servers = append(mp.servers, server)
			set.logger.Info(
				"Serving metrics",
				zap.String(zapKeyTelemetryAddress, server.Addr),
				zap.String(zapKeyTelemetryLevel, set.cfg.Level.String()),
			)
		}
		opts = append(opts, sdkmetric.WithReader(r))
	}

	var err error
	mp.MeterProvider, err = proctelemetry.InitOpenTelemetry(set.res, opts, disableHighCardinality)
	if err != nil {
		return nil, err
	}
	return mp, nil
}

// Shutdown the meter provider and all the associated resources.
// The type signature of this method matches that of the sdkmetric.MeterProvider.
func (mp *meterProvider) Shutdown(ctx context.Context) error {
	metricproducer.GlobalManager().DeleteProducer(mp.ocRegistry)

	var errs error
	for _, server := range mp.servers {
		if server != nil {
			errs = multierr.Append(errs, server.Close())
		}
	}
	return multierr.Append(errs, mp.MeterProvider.Shutdown(ctx))
}
