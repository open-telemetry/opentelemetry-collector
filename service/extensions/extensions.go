// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions // import "go.opentelemetry.io/collector/service/extensions"

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensioncapabilities"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/internal/telemetry/componentattribute"
	"go.opentelemetry.io/collector/service/internal/attribute"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/status"
	"go.opentelemetry.io/collector/service/internal/zpages"
)

const zExtensionName = "zextensionname"

// Extensions is a map of extensions created from extension configs.
type Extensions struct {
	telemetry    component.TelemetrySettings
	extMap       map[component.ID]extension.Extension
	instanceIDs  map[component.ID]*componentstatus.InstanceID
	extensionIDs []component.ID // start order (and reverse stop order)
	reporter     status.Reporter
}

// Start starts all extensions.
func (bes *Extensions) Start(ctx context.Context, host component.Host) error {
	bes.telemetry.Logger.Info("Starting extensions...")
	for _, extID := range bes.extensionIDs {
		extLogger := componentattribute.ZapLoggerWithAttributes(bes.telemetry.Logger,
			*attribute.Extension(extID).Set())
		extLogger.Info("Extension is starting...")
		instanceID := bes.instanceIDs[extID]
		ext := bes.extMap[extID]
		bes.reporter.ReportStatus(
			instanceID,
			componentstatus.NewEvent(componentstatus.StatusStarting),
		)
		if err := ext.Start(ctx, host); err != nil {
			bes.reporter.ReportStatus(
				instanceID,
				componentstatus.NewPermanentErrorEvent(err),
			)
			// We log with zap.AddStacktrace(zap.DPanicLevel) to avoid adding the stack trace to the error log
			extLogger.WithOptions(zap.AddStacktrace(zap.DPanicLevel)).Error("Failed to start extension", zap.Error(err))
			return err
		}
		bes.reporter.ReportOKIfStarting(instanceID)
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
		bes.reporter.ReportStatus(
			instanceID,
			componentstatus.NewEvent(componentstatus.StatusStopping),
		)
		if err := ext.Shutdown(ctx); err != nil {
			bes.reporter.ReportStatus(
				instanceID,
				componentstatus.NewPermanentErrorEvent(err),
			)
			errs = multierr.Append(errs, err)
			continue
		}
		bes.reporter.ReportStatus(
			instanceID,
			componentstatus.NewEvent(componentstatus.StatusStopped),
		)
	}

	return errs
}

func (bes *Extensions) NotifyPipelineReady() error {
	for _, extID := range bes.extensionIDs {
		ext := bes.extMap[extID]
		if pw, ok := ext.(extensioncapabilities.PipelineWatcher); ok {
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
		if pw, ok := ext.(extensioncapabilities.PipelineWatcher); ok {
			errs = multierr.Append(errs, pw.NotReady())
		}
	}
	return errs
}

func (bes *Extensions) NotifyConfig(ctx context.Context, conf *confmap.Conf) error {
	var errs error
	for _, extID := range bes.extensionIDs {
		ext := bes.extMap[extID]
		if cw, ok := ext.(extensioncapabilities.ConfigWatcher); ok {
			clonedConf := confmap.NewFromStringMap(conf.ToStringMap())
			errs = multierr.Append(errs, cw.NotifyConfig(ctx, clonedConf))
		}
	}
	return errs
}

func (bes *Extensions) NotifyComponentStatusChange(source *componentstatus.InstanceID, event *componentstatus.Event) {
	for _, extID := range bes.extensionIDs {
		ext := bes.extMap[extID]
		if sw, ok := ext.(componentstatus.Watcher); ok {
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
	Telemetry component.TelemetrySettings
	BuildInfo component.BuildInfo

	// Extensions builder for extensions.
	Extensions builders.Extension
}

type Option interface {
	apply(*Extensions)
}

type optionFunc func(*Extensions)

func (of optionFunc) apply(e *Extensions) {
	of(e)
}

func WithReporter(reporter status.Reporter) Option {
	return optionFunc(func(e *Extensions) {
		e.reporter = reporter
	})
}

// New creates a new Extensions from Config.
func New(ctx context.Context, set Settings, cfg Config, options ...Option) (*Extensions, error) {
	exts := &Extensions{
		telemetry:    set.Telemetry,
		extMap:       make(map[component.ID]extension.Extension),
		instanceIDs:  make(map[component.ID]*componentstatus.InstanceID),
		extensionIDs: make([]component.ID, 0, len(cfg)),
		reporter:     &nopReporter{},
	}

	for _, opt := range options {
		opt.apply(exts)
	}

	for _, extID := range cfg {
		instanceID := componentstatus.NewInstanceID(extID, component.KindExtension)
		extSet := extension.Settings{
			ID:                extID,
			TelemetrySettings: telemetry.WithAttributeSet(set.Telemetry, *attribute.Extension(extID).Set()),
			BuildInfo:         set.BuildInfo,
		}

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

type nopReporter struct{}

func (r *nopReporter) Ready() {}

func (r *nopReporter) ReportStatus(*componentstatus.InstanceID, *componentstatus.Event) {}

func (r *nopReporter) ReportOKIfStarting(*componentstatus.InstanceID) {}
