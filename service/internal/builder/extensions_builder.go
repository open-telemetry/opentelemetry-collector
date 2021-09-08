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

package builder

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumererror"
)

// builtExporter is an exporter that is built based on a config. It can have
// a trace and/or a metrics consumer and have a shutdown function.
type builtExtension struct {
	logger    *zap.Logger
	extension component.Extension
}

// Start the receiver.
func (ext *builtExtension) Start(ctx context.Context, host component.Host) error {
	return ext.extension.Start(ctx, host)
}

// Shutdown the receiver.
func (ext *builtExtension) Shutdown(ctx context.Context) error {
	return ext.extension.Shutdown(ctx)
}

var _ component.Extension = (*builtExtension)(nil)

// Extensions is a map of extensions created from extension configs.
type Extensions map[config.ComponentID]*builtExtension

// StartAll starts all exporters.
func (exts Extensions) StartAll(ctx context.Context, host component.Host) error {
	for _, ext := range exts {
		ext.logger.Info("Extension is starting...")

		if err := ext.Start(ctx, newHostWrapper(host, ext.logger)); err != nil {
			return err
		}

		ext.logger.Info("Extension started.")
	}
	return nil
}

// ShutdownAll stops all exporters.
func (exts Extensions) ShutdownAll(ctx context.Context) error {
	var errs []error
	for _, ext := range exts {
		err := ext.Shutdown(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return consumererror.Combine(errs)
}

func (exts Extensions) NotifyPipelineReady() error {
	for _, ext := range exts {
		if pw, ok := ext.extension.(component.PipelineWatcher); ok {
			if err := pw.Ready(); err != nil {
				ext.logger.Error("Error notifying extension that the pipeline was started.")
				return err
			}
		}
	}

	return nil
}

func (exts Extensions) NotifyPipelineNotReady() error {
	// Notify extensions in reverse order.
	var errs []error
	for _, ext := range exts {
		if pw, ok := ext.extension.(component.PipelineWatcher); ok {
			if err := pw.NotReady(); err != nil {
				ext.logger.Error("Error notifying extension that the pipeline was shutdown.")
				errs = append(errs, err)
			}
		}
	}

	return consumererror.Combine(errs)
}

func (exts Extensions) ToMap() map[config.ComponentID]component.Extension {
	result := make(map[config.ComponentID]component.Extension, len(exts))
	for extID, v := range exts {
		result[extID] = v.extension
	}
	return result
}

// BuildExtensions builds Extensions from config.
func BuildExtensions(
	logger *zap.Logger,
	tracerProvider trace.TracerProvider,
	buildInfo component.BuildInfo,
	config *config.Config,
	factories map[config.Type]component.ExtensionFactory,
) (Extensions, error) {
	extensions := make(Extensions)
	for _, extID := range config.Service.Extensions {
		extCfg, existsCfg := config.Extensions[extID]
		if !existsCfg {
			return nil, fmt.Errorf("extension %q is not configured", extID)
		}

		factory, existsFactory := factories[extID.Type()]
		if !existsFactory {
			return nil, fmt.Errorf("extension factory for type %q is not configured", extID.Type())
		}

		set := component.ExtensionCreateSettings{
			TelemetryCreateSettings: component.TelemetryCreateSettings{
				Logger: logger.With(
					zap.String(zapKindKey, zapKindExtension),
					zap.String(zapNameKey, extID.String())),
				TracerProvider: tracerProvider,
			},
			BuildInfo: buildInfo,
		}
		ext, err := buildExtension(context.Background(), factory, set, extCfg)
		if err != nil {
			return nil, err
		}

		extensions[extID] = ext
	}

	return extensions, nil
}

func buildExtension(ctx context.Context, factory component.ExtensionFactory, creationSet component.ExtensionCreateSettings, cfg config.Extension) (*builtExtension, error) {
	ext := &builtExtension{
		logger: creationSet.Logger,
	}

	ex, err := factory.CreateExtension(ctx, creationSet, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create extension %v: %w", cfg.ID(), err)
	}

	// Check if the factory really created the extension.
	if ex == nil {
		return nil, fmt.Errorf("factory for %v produced a nil extension", cfg.ID())
	}

	ext.extension = ex

	return ext, nil
}
