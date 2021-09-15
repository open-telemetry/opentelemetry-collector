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

package service

import (
	"context"
	"fmt"

	"go.opentelemetry.io/contrib/zpages"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/service/internal/builder"
)

// service represents the implementation of a component.Host.
type service struct {
	factories           component.Factories
	buildInfo           component.BuildInfo
	config              *config.Config
	logger              *zap.Logger
	tracerProvider      trace.TracerProvider
	meterProvider       metric.MeterProvider
	zPagesSpanProcessor *zpages.SpanProcessor
	asyncErrorChannel   chan error

	builtExporters  builder.Exporters
	builtReceivers  builder.Receivers
	builtPipelines  builder.BuiltPipelines
	builtExtensions builder.Extensions
}

func newService(set *svcSettings) (*service, error) {
	srv := &service{
		factories:           set.Factories,
		buildInfo:           set.BuildInfo,
		config:              set.Config,
		logger:              set.Logger,
		tracerProvider:      set.TracerProvider,
		meterProvider:       set.MeterProvider,
		zPagesSpanProcessor: set.ZPagesSpanProcessor,
		asyncErrorChannel:   set.AsyncErrorChannel,
	}

	if err := srv.config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	var err error
	telemetrySettings := component.TelemetrySettings{
		Logger:         srv.logger,
		TracerProvider: srv.tracerProvider,
		MeterProvider:  srv.meterProvider,
	}
	srv.builtExtensions, err = builder.BuildExtensions(telemetrySettings, srv.buildInfo, srv.config, srv.factories.Extensions)
	if err != nil {
		return nil, fmt.Errorf("cannot build extensions: %w", err)
	}

	// Pipeline is built backwards, starting from exporters, so that we create objects
	// which are referenced before objects which reference them.

	// First create exporters.
	srv.builtExporters, err = builder.BuildExporters(telemetrySettings, srv.buildInfo, srv.config, srv.factories.Exporters)
	if err != nil {
		return nil, fmt.Errorf("cannot build exporters: %w", err)
	}

	// Create pipelines and their processors and plug exporters to the end of the pipelines.
	srv.builtPipelines, err = builder.BuildPipelines(telemetrySettings, srv.buildInfo, srv.config, srv.builtExporters, srv.factories.Processors)
	if err != nil {
		return nil, fmt.Errorf("cannot build pipelines: %w", err)
	}

	// Create receivers and plug them into the start of the pipelines.
	srv.builtReceivers, err = builder.BuildReceivers(telemetrySettings, srv.buildInfo, srv.config, srv.builtPipelines, srv.factories.Receivers)
	if err != nil {
		return nil, fmt.Errorf("cannot build receivers: %w", err)
	}

	return srv, nil
}

func (srv *service) Start(ctx context.Context) error {
	srv.logger.Info("Starting extensions...")
	if err := srv.builtExtensions.StartAll(ctx, srv); err != nil {
		return fmt.Errorf("failed to start extensions: %w", err)
	}

	srv.logger.Info("Starting exporters...")
	if err := srv.builtExporters.StartAll(ctx, srv); err != nil {
		return fmt.Errorf("cannot start exporters: %w", err)
	}

	srv.logger.Info("Starting processors...")
	if err := srv.builtPipelines.StartProcessors(ctx, srv); err != nil {
		return fmt.Errorf("cannot start processors: %w", err)
	}

	srv.logger.Info("Starting receivers...")
	if err := srv.builtReceivers.StartAll(ctx, srv); err != nil {
		return fmt.Errorf("cannot start receivers: %w", err)
	}

	return srv.builtExtensions.NotifyPipelineReady()
}

func (srv *service) Shutdown(ctx context.Context) error {
	// Accumulate errors and proceed with shutting down remaining components.
	var errs []error

	if err := srv.builtExtensions.NotifyPipelineNotReady(); err != nil {
		errs = append(errs, fmt.Errorf("failed to notify that pipeline is not ready: %w", err))
	}

	// Pipeline shutdown order is the reverse of building/starting: first receivers, then flushing pipelines
	// giving senders a chance to send all their data. This may take time, the allowed
	// time should be part of configuration.

	srv.logger.Info("Stopping receivers...")
	if err := srv.builtReceivers.ShutdownAll(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown receivers: %w", err))
	}

	srv.logger.Info("Stopping processors...")
	if err := srv.builtPipelines.ShutdownProcessors(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown processors: %w", err))
	}

	srv.logger.Info("Stopping exporters...")
	if err := srv.builtExporters.ShutdownAll(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown exporters: %w", err))
	}

	srv.logger.Info("Stopping extensions...")
	if err := srv.builtExtensions.ShutdownAll(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown extensions: %w", err))
	}

	return consumererror.Combine(errs)
}

// ReportFatalError is used to report to the host that the receiver encountered
// a fatal error (i.e.: an error that the instance can't recover from) after
// its start function has already returned.
func (srv *service) ReportFatalError(err error) {
	srv.asyncErrorChannel <- err
}

func (srv *service) GetFactory(kind component.Kind, componentType config.Type) component.Factory {
	switch kind {
	case component.KindReceiver:
		return srv.factories.Receivers[componentType]
	case component.KindProcessor:
		return srv.factories.Processors[componentType]
	case component.KindExporter:
		return srv.factories.Exporters[componentType]
	case component.KindExtension:
		return srv.factories.Extensions[componentType]
	}
	return nil
}

func (srv *service) GetExtensions() map[config.ComponentID]component.Extension {
	return srv.builtExtensions.ToMap()
}

func (srv *service) GetExporters() map[config.DataType]map[config.ComponentID]component.Exporter {
	return srv.builtExporters.ToMapByDataType()
}
