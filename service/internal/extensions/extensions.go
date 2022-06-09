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

package extensions // import "go.opentelemetry.io/collector/service/internal/extensions"

import (
	"context"
	"fmt"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/service/internal/components"
)

// BuiltExtensions is a map of extensions created from extension configs.
type BuiltExtensions struct {
	settings component.TelemetrySettings
	extMap   map[config.ComponentID]component.Extension
}

// StartAll starts all extensions.
func (exts BuiltExtensions) StartAll(ctx context.Context, host component.Host) error {
	for extID, ext := range exts.extMap {
		extLogger := extensionLogger(exts.settings.Logger, extID)
		extLogger.Info("Extension is starting...")
		if err := ext.Start(ctx, components.NewHostWrapper(host, extLogger)); err != nil {
			return err
		}
		extLogger.Info("Extension started.")
	}
	return nil
}

// ShutdownAll stops all extensions.
func (exts BuiltExtensions) ShutdownAll(ctx context.Context) error {
	var errs error
	for _, ext := range exts.extMap {
		errs = multierr.Append(errs, ext.Shutdown(ctx))
	}

	return errs
}

func (exts BuiltExtensions) NotifyPipelineReady() error {
	for extID, ext := range exts.extMap {
		if pw, ok := ext.(component.PipelineWatcher); ok {
			if err := pw.Ready(); err != nil {
				extensionLogger(exts.settings.Logger, extID).Error("Error notifying extension that the pipeline was started.")
				return err
			}
		}
	}
	return nil
}

func (exts BuiltExtensions) NotifyPipelineNotReady() error {
	// Notify extensions in reverse order.
	var errs error
	for _, ext := range exts.extMap {
		if pw, ok := ext.(component.PipelineWatcher); ok {
			errs = multierr.Append(errs, pw.NotReady())
		}
	}
	return errs
}

func (exts BuiltExtensions) ToMap() map[config.ComponentID]component.Extension {
	result := make(map[config.ComponentID]component.Extension, len(exts.extMap))
	for extID, v := range exts.extMap {
		result[extID] = v
	}
	return result
}

// Build builds BuiltExtensions from config.
func Build(
	ctx context.Context,
	settings component.TelemetrySettings,
	buildInfo component.BuildInfo,
	extensionsConfigs map[config.ComponentID]config.Extension,
	serviceExtensions []config.ComponentID,
	factories map[config.Type]component.ExtensionFactory,
) (*BuiltExtensions, error) {
	exts := &BuiltExtensions{
		settings: settings,
		extMap:   make(map[config.ComponentID]component.Extension),
	}
	for _, extID := range serviceExtensions {
		extCfg, existsCfg := extensionsConfigs[extID]
		if !existsCfg {
			return nil, fmt.Errorf("extension %q is not configured", extID)
		}

		factory, existsFactory := factories[extID.Type()]
		if !existsFactory {
			return nil, fmt.Errorf("extension factory for type %q is not configured", extID.Type())
		}

		set := component.ExtensionCreateSettings{
			TelemetrySettings: settings,
			BuildInfo:         buildInfo,
		}
		set.TelemetrySettings.Logger = settings.Logger
		ext, err := buildExtension(ctx, factory, set, extCfg)
		if err != nil {
			return nil, err
		}

		exts.extMap[extID] = ext
	}

	return exts, nil
}

func extensionLogger(logger *zap.Logger, id config.ComponentID) *zap.Logger {
	return logger.With(
		zap.String(components.ZapKindKey, components.ZapKindExtension),
		zap.String(components.ZapNameKey, id.String()))
}

func buildExtension(ctx context.Context, factory component.ExtensionFactory, creationSet component.ExtensionCreateSettings, cfg config.Extension) (component.Extension, error) {
	ext, err := factory.CreateExtension(ctx, creationSet, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create extension %q: %w", cfg.ID(), err)
	}

	// Check if the factory really created the extension.
	if ext == nil {
		return nil, fmt.Errorf("factory for %q produced a nil extension", cfg.ID())
	}

	return ext, nil
}
