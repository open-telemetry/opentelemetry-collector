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
	"net/http"
	"sort"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/service/internal/components"
	"go.opentelemetry.io/collector/service/internal/zpages"
)

const zExtensionName = "zextensionname"

// BuiltExtensions is a map of extensions created from extension configs.
type BuiltExtensions struct {
	telemetry component.TelemetrySettings
	extMap    map[config.ComponentID]component.Extension
}

// StartAll starts all extensions.
func (bes *BuiltExtensions) StartAll(ctx context.Context, host component.Host) error {
	bes.telemetry.Logger.Info("Starting extensions...")
	for extID, ext := range bes.extMap {
		extLogger := extensionLogger(bes.telemetry.Logger, extID)
		extLogger.Info("Extension is starting...")
		if err := ext.Start(ctx, components.NewHostWrapper(host, extLogger)); err != nil {
			return err
		}
		extLogger.Info("Extension started.")
	}
	return nil
}

// ShutdownAll stops all extensions.
func (bes *BuiltExtensions) ShutdownAll(ctx context.Context) error {
	bes.telemetry.Logger.Info("Stopping extensions...")
	var errs error
	for _, ext := range bes.extMap {
		errs = multierr.Append(errs, ext.Shutdown(ctx))
	}

	return errs
}

func (bes *BuiltExtensions) NotifyPipelineReady() error {
	for extID, ext := range bes.extMap {
		if pw, ok := ext.(component.PipelineWatcher); ok {
			if err := pw.Ready(); err != nil {
				return fmt.Errorf("failed to notify extension %q: %w", extID, err)
			}
		}
	}
	return nil
}

func (bes *BuiltExtensions) NotifyPipelineNotReady() error {
	// Notify extensions in reverse order.
	var errs error
	for _, ext := range bes.extMap {
		if pw, ok := ext.(component.PipelineWatcher); ok {
			errs = multierr.Append(errs, pw.NotReady())
		}
	}
	return errs
}

func (bes *BuiltExtensions) GetExtensions() map[config.ComponentID]component.Extension {
	result := make(map[config.ComponentID]component.Extension, len(bes.extMap))
	for extID, v := range bes.extMap {
		result[extID] = v
	}
	return result
}

func (bes *BuiltExtensions) HandleZPages(w http.ResponseWriter, r *http.Request) {
	extensionName := r.URL.Query().Get(zExtensionName)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	zpages.WriteHTMLPageHeader(w, zpages.HeaderData{Title: "Extensions"})
	data := zpages.SummaryExtensionsTableData{}

	data.Rows = make([]zpages.SummaryExtensionsTableRowData, 0, len(bes.extMap))
	for id := range bes.extMap {
		row := zpages.SummaryExtensionsTableRowData{FullName: id.String()}
		data.Rows = append(data.Rows, row)
	}

	sort.Slice(data.Rows, func(i, j int) bool {
		return data.Rows[i].FullName < data.Rows[j].FullName
	})
	zpages.WriteHTMLExtensionsSummaryTable(w, data)
	if extensionName != "" {
		zpages.WriteHTMLComponentHeader(w, zpages.ComponentHeaderData{
			Name: extensionName,
		})
		// TODO: Add config + status info.
	}
	zpages.WriteHTMLPageFooter(w)
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
		telemetry: settings,
		extMap:    make(map[config.ComponentID]component.Extension),
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
