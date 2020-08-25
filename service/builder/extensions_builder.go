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

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/config/configmodels"
)

// builtExporter is an exporter that is built based on a config. It can have
// a trace and/or a metrics consumer and have a shutdown function.
type builtExtension struct {
	logger    *zap.Logger
	extension component.ServiceExtension
}

// Start the receiver.
func (ext *builtExtension) Start(ctx context.Context, host component.Host) error {
	return ext.extension.Start(ctx, host)
}

// Stop the receiver.
func (ext *builtExtension) Shutdown(ctx context.Context) error {
	return ext.extension.Shutdown(ctx)
}

var _ component.ServiceExtension = (*builtExtension)(nil)

// Exporters is a map of exporters created from exporter configs.
type Extensions map[configmodels.Extension]*builtExtension

// StartAll starts all exporters.
func (exts Extensions) StartAll(ctx context.Context, host component.Host) error {
	for _, ext := range exts {
		ext.logger.Info("Extension is starting...")

		if err := ext.Start(ctx, host); err != nil {
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

	return componenterror.CombineErrors(errs)
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

	return componenterror.CombineErrors(errs)
}

func (exts Extensions) ToMap() map[configmodels.Extension]component.ServiceExtension {
	result := make(map[configmodels.Extension]component.ServiceExtension, len(exts))
	for k, v := range exts {
		result[k] = v.extension
	}
	return result
}

// ExportersBuilder builds exporters from config.
type ExtensionsBuilder struct {
	logger    *zap.Logger
	appInfo   component.ApplicationStartInfo
	config    *configmodels.Config
	factories map[configmodels.Type]component.ExtensionFactory
}

// NewExportersBuilder creates a new ExportersBuilder. Call BuildExporters() on the returned value.
func NewExtensionsBuilder(
	logger *zap.Logger,
	appInfo component.ApplicationStartInfo,
	config *configmodels.Config,
	factories map[configmodels.Type]component.ExtensionFactory,
) *ExtensionsBuilder {
	return &ExtensionsBuilder{logger.With(zap.String(kindLogKey, kindLogExtension)), appInfo, config, factories}
}

// Build extensions from config.
func (eb *ExtensionsBuilder) Build() (Extensions, error) {
	extensions := make(Extensions)

	for _, extName := range eb.config.Service.Extensions {
		extCfg, exists := eb.config.Extensions[extName]
		if !exists {
			return nil, fmt.Errorf("extension %q is not configured", extName)
		}

		componentLogger := eb.logger.With(zap.String(typeLogKey, string(extCfg.Type())), zap.String(nameLogKey, extCfg.Name()))
		ext, err := eb.buildExtension(componentLogger, eb.appInfo, extCfg)
		if err != nil {
			return nil, err
		}

		extensions[extCfg] = ext
	}

	return extensions, nil
}

func (eb *ExtensionsBuilder) buildExtension(logger *zap.Logger, appInfo component.ApplicationStartInfo, cfg configmodels.Extension) (*builtExtension, error) {
	factory := eb.factories[cfg.Type()]
	if factory == nil {
		return nil, fmt.Errorf("extension factory for type %q is not configured", cfg.Type())
	}

	ext := &builtExtension{
		logger: logger,
	}

	creationParams := component.ExtensionCreateParams{
		Logger:               logger,
		ApplicationStartInfo: appInfo,
	}

	ex, err := factory.CreateExtension(context.Background(), creationParams, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create extension %q: %w", cfg.Name(), err)
	}

	// Check if the factory really created the extension.
	if ex == nil {
		return nil, fmt.Errorf("factory for %q produced a nil extension", cfg.Name())
	}

	ext.extension = ex

	return ext, nil
}
