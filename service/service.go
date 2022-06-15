// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service // import "go.opentelemetry.io/collector/service"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/metric/nonrecording"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/internal"
	"go.opentelemetry.io/collector/service/internal/extensions"
	"go.opentelemetry.io/collector/service/internal/pipelines"
	"go.opentelemetry.io/collector/service/internal/telemetrylogs"
)

// service represents the implementation of a component.Host.
type service struct {
	buildInfo component.BuildInfo
	config    *Config
	telemetry component.TelemetrySettings
	host      *serviceHost
}

func newService(set *settings) (*service, error) {
	srv := &service{
		buildInfo: set.BuildInfo,
		config:    set.Config,
		telemetry: component.TelemetrySettings{
			TracerProvider: trace.NewNoopTracerProvider(),
			MeterProvider:  nonrecording.NewNoopMeterProvider(),
			MetricsLevel:   set.Config.Telemetry.Metrics.Level,
		},
		host: &serviceHost{
			factories:         set.Factories,
			buildInfo:         set.BuildInfo,
			asyncErrorChannel: set.AsyncErrorChannel,
		},
	}

	srv.telemetry.TracerProvider = sdktrace.NewTracerProvider(
		// needed for supporting the zpages extension
		sdktrace.WithSampler(internal.AlwaysRecord()),
	)

	var err error
	if srv.telemetry.Logger, err = telemetrylogs.NewLogger(set.Config.Service.Telemetry.Logs, set.LoggingOptions); err != nil {
		return nil, fmt.Errorf("failed to get logger: %w", err)
	}

	if srv.host.builtExtensions, err = extensions.Build(context.Background(), srv.telemetry, srv.buildInfo, srv.config.Extensions, srv.config.Service.Extensions, srv.host.factories.Extensions); err != nil {
		return nil, fmt.Errorf("cannot build extensions: %w", err)
	}

	if srv.host.pipelines, err = pipelines.Build(context.Background(), srv.telemetry, srv.buildInfo, srv.config, srv.host.factories); err != nil {
		return nil, fmt.Errorf("cannot build pipelines: %w", err)
	}

	return srv, nil
}

func (srv *service) Start(ctx context.Context) error {
	if err := srv.host.builtExtensions.StartAll(ctx, srv.host); err != nil {
		return fmt.Errorf("failed to start extensions: %w", err)
	}

	if err := srv.host.pipelines.StartAll(ctx, srv.host); err != nil {
		return fmt.Errorf("cannot start exporters: %w", err)
	}

	return srv.host.builtExtensions.NotifyPipelineReady()
}

func (srv *service) Shutdown(ctx context.Context) error {
	// Accumulate errors and proceed with shutting down remaining components.
	var errs error

	if err := srv.host.builtExtensions.NotifyPipelineNotReady(); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to notify that pipeline is not ready: %w", err))
	}

	if err := srv.host.pipelines.ShutdownAll(ctx); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown exporters: %w", err))
	}

	if err := srv.host.builtExtensions.ShutdownAll(ctx); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown extensions: %w", err))
	}

	// TODO: Shutdown TracerProvider, MeterProvider, and Sync Logger.
	return errs
}
