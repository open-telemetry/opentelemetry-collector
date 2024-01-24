// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions // import "go.opentelemetry.io/collector/service/extensions"

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/service/internal/components"
	"go.opentelemetry.io/collector/service/internal/servicetelemetry"
	"go.opentelemetry.io/collector/service/internal/zpages"
)

const zExtensionName = "zextensionname"

// Extensions is a map of extensions created from extension configs.
type Extensions struct {
	telemetry    servicetelemetry.TelemetrySettings
	extMap       map[component.ID]extension.Extension
	instanceIDs  map[component.ID]*component.InstanceID
	extensionIDs []component.ID // start order (and reverse stop order)
}

// Start starts all extensions.
func (bes *Extensions) Start(ctx context.Context, host component.Host) error {
	bes.telemetry.Logger.Info("Starting extensions...")
	for _, extID := range bes.extensionIDs {
		extLogger := components.ExtensionLogger(bes.telemetry.Logger, extID)
		extLogger.Info("Extension is starting...")
		instanceID := bes.instanceIDs[extID]
		ext := bes.extMap[extID]
		bes.telemetry.Status.ReportStatus(
			instanceID,
			component.NewStatusEvent(component.StatusStarting),
		)
		if err := ext.Start(ctx, host); err != nil {
			bes.telemetry.Status.ReportStatus(
				instanceID,
				component.NewPermanentErrorEvent(err),
			)
			return err
		}
		bes.telemetry.Status.ReportOKIfStarting(instanceID)
		extLogger.Info("Extension started.")
	}
	return nil
}

// Shutdown stops all extensions.
func (bes *Extensions) Shutdown(ctx context.Context) error {
	bes.telemetry.Logger.Info("Stopping extensions...")
	var errs error
	for i := len(bes.extensionIDs) - 1; i >= 0; i-- {
		extID := bes.extensionIDs[i]
		instanceID := bes.instanceIDs[extID]
		ext := bes.extMap[extID]
		bes.telemetry.Status.ReportStatus(
			instanceID,
			component.NewStatusEvent(component.StatusStopping),
		)
		if err := ext.Shutdown(ctx); err != nil {
			bes.telemetry.Status.ReportStatus(
				instanceID,
				component.NewPermanentErrorEvent(err),
			)
			errs = multierr.Append(errs, err)
			continue
		}
		bes.telemetry.Status.ReportStatus(
			instanceID,
			component.NewStatusEvent(component.StatusStopped),
		)
	}

	return errs
}

func (bes *Extensions) NotifyPipelineReady() error {
	for _, extID := range bes.extensionIDs {
		ext := bes.extMap[extID]
		if pw, ok := ext.(extension.PipelineWatcher); ok {
			if err := pw.Ready(); err != nil {
				return fmt.Errorf("failed to notify extension %q: %w", extID, err)
			}
		}
	}
	return nil
}

func (bes *Extensions) NotifyPipelineNotReady() error {
	var errs error
	for _, extID := range bes.extensionIDs {
		ext := bes.extMap[extID]
		if pw, ok := ext.(extension.PipelineWatcher); ok {
			errs = multierr.Append(errs, pw.NotReady())
		}
	}
	return errs
}

func (bes *Extensions) NotifyConfig(ctx context.Context, conf *confmap.Conf) error {
	var errs error
	for _, extID := range bes.extensionIDs {
		ext := bes.extMap[extID]
		if cw, ok := ext.(extension.ConfigWatcher); ok {
			clonedConf := confmap.NewFromStringMap(conf.ToStringMap())
			errs = multierr.Append(errs, cw.NotifyConfig(ctx, clonedConf))
		}
	}
	return errs
}

func (bes *Extensions) NotifyComponentStatusChange(source *component.InstanceID, event *component.StatusEvent) {
	for _, extID := range bes.extensionIDs {
		ext := bes.extMap[extID]
		if sw, ok := ext.(extension.StatusWatcher); ok {
			sw.ComponentStatusChanged(source, event)
		}
	}
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
	for _, id := range bes.extensionIDs {
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
	Telemetry servicetelemetry.TelemetrySettings
	BuildInfo component.BuildInfo

	// Extensions builder for extensions.
	Extensions *extension.Builder
}

// New creates a new Extensions from Config.
func New(ctx context.Context, set Settings, cfg Config) (*Extensions, error) {
	exts := &Extensions{
		telemetry:    set.Telemetry,
		extMap:       make(map[component.ID]extension.Extension),
		instanceIDs:  make(map[component.ID]*component.InstanceID),
		extensionIDs: make([]component.ID, 0, len(cfg)),
	}
	for _, extID := range cfg {
		instanceID := &component.InstanceID{
			ID:   extID,
			Kind: component.KindExtension,
		}
		extSet := extension.CreateSettings{
			ID:                extID,
			TelemetrySettings: set.Telemetry.ToComponentTelemetrySettings(instanceID),
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
		exts.instanceIDs[extID] = instanceID
	}
	order, err := computeOrder(exts)
	if err != nil {
		return nil, err
	}
	exts.extensionIDs = order
	return exts, nil
}
