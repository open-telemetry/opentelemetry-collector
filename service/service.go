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

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/service/internal/builder"
	"go.opentelemetry.io/collector/service/internal/extensions"
)

// service represents the implementation of a component.Host.
type service struct {
	buildInfo component.BuildInfo
	config    *config.Config
	telemetry component.TelemetrySettings
	host      *serviceHost
}

func newService(set *svcSettings) (*service, error) {
	srv := &service{
		buildInfo: set.BuildInfo,
		config:    set.Config,
		telemetry: set.Telemetry,
		host: &serviceHost{
			factories:         set.Factories,
			asyncErrorChannel: set.AsyncErrorChannel,
		},
	}

	var err error
	if srv.host.builtExtensions, err = extensions.Build(srv.telemetry, srv.buildInfo, srv.config, srv.host.factories.Extensions); err != nil {
		return nil, fmt.Errorf("cannot build extensions: %w", err)
	}

	// Pipeline is built backwards, starting from exporters, so that we create objects
	// which are referenced before objects which reference them.

	// First create exporters.
	if srv.host.builtExporters, err = builder.BuildExporters(srv.telemetry, srv.buildInfo, srv.config, srv.host.factories.Exporters); err != nil {
		return nil, fmt.Errorf("cannot build exporters: %w", err)
	}

	// Create pipelines and their processors and plug exporters to the end of the pipelines.
	if srv.host.builtPipelines, err = builder.BuildPipelines(srv.telemetry, srv.buildInfo, srv.config, srv.host.builtExporters, srv.host.factories.Processors); err != nil {
		return nil, fmt.Errorf("cannot build pipelines: %w", err)
	}

	// Create receivers and plug them into the start of the pipelines.
	if srv.host.builtReceivers, err = builder.BuildReceivers(srv.telemetry, srv.buildInfo, srv.config, srv.host.builtPipelines, srv.host.factories.Receivers); err != nil {
		return nil, fmt.Errorf("cannot build receivers: %w", err)
	}

	return srv, nil
}

func (srv *service) Start(ctx context.Context) error {
	srv.telemetry.Logger.Info("Starting extensions...")
	if err := srv.host.builtExtensions.StartAll(ctx, srv.host); err != nil {
		return fmt.Errorf("failed to start extensions: %w", err)
	}

	srv.telemetry.Logger.Info("Starting exporters...")
	if err := srv.host.builtExporters.StartAll(ctx, srv.host); err != nil {
		return fmt.Errorf("cannot start exporters: %w", err)
	}

	srv.telemetry.Logger.Info("Starting processors...")
	if err := srv.host.builtPipelines.StartProcessors(ctx, srv.host); err != nil {
		return fmt.Errorf("cannot start processors: %w", err)
	}

	srv.telemetry.Logger.Info("Starting receivers...")
	if err := srv.host.builtReceivers.StartAll(ctx, srv.host); err != nil {
		return fmt.Errorf("cannot start receivers: %w", err)
	}

	return srv.host.builtExtensions.NotifyPipelineReady()
}

func (srv *service) Shutdown(ctx context.Context) error {
	// Accumulate errors and proceed with shutting down remaining components.
	var errs error

	if err := srv.host.builtExtensions.NotifyPipelineNotReady(); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to notify that pipeline is not ready: %w", err))
	}

	// Pipeline shutdown order is the reverse of building/starting: first receivers, then flushing pipelines
	// giving senders a chance to send all their data. This may take time, the allowed
	// time should be part of configuration.

	srv.telemetry.Logger.Info("Stopping receivers...")
	if err := srv.host.builtReceivers.ShutdownAll(ctx); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown receivers: %w", err))
	}

	srv.telemetry.Logger.Info("Stopping processors...")
	if err := srv.host.builtPipelines.ShutdownProcessors(ctx); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown processors: %w", err))
	}

	srv.telemetry.Logger.Info("Stopping exporters...")
	if err := srv.host.builtExporters.ShutdownAll(ctx); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown exporters: %w", err))
	}

	srv.telemetry.Logger.Info("Stopping extensions...")
	if err := srv.host.builtExtensions.ShutdownAll(ctx); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("failed to shutdown extensions: %w", err))
	}

	return errs
}
