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

package extensions // import "go.opentelemetry.io/collector/service/extensions"

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/service/internal/components"
	"go.opentelemetry.io/collector/service/internal/zpages"
)

const zExtensionName = "zextensionname"

// Extensions is a map of extensions created from extension configs.
type Extensions struct {
	telemetry component.TelemetrySettings
	extMap    map[component.ID]extension.Extension
}

// Start starts all extensions.
func (bes *Extensions) Start(ctx context.Context, host component.Host) error {
	bes.telemetry.Logger.Info("Starting extensions...")
	for extID, ext := range bes.extMap {
		extLogger := components.ExtensionLogger(bes.telemetry.Logger, extID)
		extLogger.Info("Extension is starting...")
		if err := ext.Start(ctx, components.NewHostWrapper(host, extLogger)); err != nil {
			return err
		}
		extLogger.Info("Extension started.")
	}
	return nil
}

// Shutdown stops all extensions.
func (bes *Extensions) Shutdown(ctx context.Context) error {
	bes.telemetry.Logger.Info("Stopping extensions...")
	var errs error
	for _, ext := range bes.extMap {
		errs = multierr.Append(errs, ext.Shutdown(ctx))
	}

	return errs
}

func (bes *Extensions) NotifyPipelineReady() error {
	for extID, ext := range bes.extMap {
		if pw, ok := ext.(extension.PipelineWatcher); ok {
			if err := pw.Ready(); err != nil {
				return fmt.Errorf("failed to notify extension %q: %w", extID, err)
			}
		}
	}
	return nil
}

func (bes *Extensions) NotifyPipelineNotReady() error {
	// Notify extensions in reverse order.
	var errs error
	for _, ext := range bes.extMap {
		if pw, ok := ext.(extension.PipelineWatcher); ok {
			errs = multierr.Append(errs, pw.NotReady())
		}
	}
	return errs
}

func (bes *Extensions) GetExtensions() map[component.ID]component.Component {
	result := make(map[component.ID]component.Component, len(bes.extMap))
	for extID, v := range bes.extMap {
		result[extID] = v
	}
	return result
}

func (bes *Extensions) HandleZPages(w http.ResponseWriter, r *http.Request) {
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

// Settings holds configuration for building Extensions.
type Settings struct {
	Telemetry component.TelemetrySettings
	BuildInfo component.BuildInfo

	// Drepecated: [v0.68.0] use Extensions.
	Configs map[component.ID]component.Config

	// Drepecated: [v0.68.0] use Extensions.
	Factories map[component.Type]extension.Factory

	// Extensions builder for extensions.
	Extensions *extension.Builder
}

// New creates a new Extensions from Config.
func New(ctx context.Context, set Settings, cfg Config) (*Extensions, error) {
	if set.Extensions == nil {
		set.Extensions = extension.NewBuilder(set.Configs, set.Factories)
	}
	exts := &Extensions{
		telemetry: set.Telemetry,
		extMap:    make(map[component.ID]extension.Extension),
	}
	for _, extID := range cfg {
		extSet := extension.CreateSettings{
			ID:                extID,
			TelemetrySettings: set.Telemetry,
			BuildInfo:         set.BuildInfo,
		}
		extSet.TelemetrySettings.Logger = components.ExtensionLogger(set.Telemetry.Logger, extID)

		ext, err := set.Extensions.Create(ctx, extSet)
		if err != nil {
			return nil, fmt.Errorf("failed to create extension %q: %w", extID, err)
		}

		// Check if the factory really created the extension.
		if ext == nil {
			return nil, fmt.Errorf("factory for %q produced a nil extension", extID)
		}

		exts.extMap[extID] = ext
	}

	return exts, nil
}
