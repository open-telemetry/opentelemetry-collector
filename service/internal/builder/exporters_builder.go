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

package builder // import "go.opentelemetry.io/collector/service/internal/builder"

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/service/internal/components"
)

// BuiltExporters is a map of exporters created from exporter configs.
type BuiltExporters struct {
	settings  component.TelemetrySettings
	exporters map[config.DataType]map[config.ComponentID]component.Exporter
}

// StartAll starts all exporters.
func (exps BuiltExporters) StartAll(ctx context.Context, host component.Host) error {
	for dt, expByID := range exps.exporters {
		for expID, exp := range expByID {
			expLogger := exporterLogger(exps.settings.Logger, expID, dt)
			expLogger.Info("Exporter is starting...")
			if err := exp.Start(ctx, components.NewHostWrapper(host, expLogger)); err != nil {
				return err
			}
			expLogger.Info("Exporter started.")
		}
	}
	return nil
}

// ShutdownAll stops all exporters.
func (exps BuiltExporters) ShutdownAll(ctx context.Context) error {
	var errs error
	for _, expByID := range exps.exporters {
		for _, exp := range expByID {
			errs = multierr.Append(errs, exp.Shutdown(ctx))
		}
	}
	return errs
}

func (exps BuiltExporters) ToMapByDataType() map[config.DataType]map[config.ComponentID]component.Exporter {
	exportersMap := make(map[config.DataType]map[config.ComponentID]component.Exporter)

	exportersMap[config.TracesDataType] = make(map[config.ComponentID]component.Exporter, len(exps.exporters[config.TracesDataType]))
	exportersMap[config.MetricsDataType] = make(map[config.ComponentID]component.Exporter, len(exps.exporters[config.MetricsDataType]))
	exportersMap[config.LogsDataType] = make(map[config.ComponentID]component.Exporter, len(exps.exporters[config.LogsDataType]))

	for dt, expByID := range exps.exporters {
		for expID, exp := range expByID {
			exportersMap[dt][expID] = exp
		}
	}

	return exportersMap
}

// BuildExporters builds Exporters from config.
func BuildExporters(
	ctx context.Context,
	settings component.TelemetrySettings,
	buildInfo component.BuildInfo,
	cfg *config.Config,
	factories map[config.Type]component.ExporterFactory,
) (*BuiltExporters, error) {
	exps := &BuiltExporters{
		settings:  settings,
		exporters: make(map[config.DataType]map[config.ComponentID]component.Exporter),
	}

	// Go over all pipelines. The data type of the pipeline defines what data type
	// each exporter is expected to receive.

	// Iterate over pipelines.
	for pipelineID, pipeline := range cfg.Service.Pipelines {
		dt := pipelineID.Type()
		if _, ok := exps.exporters[dt]; !ok {
			exps.exporters[dt] = make(map[config.ComponentID]component.Exporter)
		}
		expByID := exps.exporters[dt]

		// Iterate over all exporters for this pipeline.
		for _, expID := range pipeline.Exporters {
			// If already created an exporter for this [DataType, ComponentID] nothing to do, will reuse this instance.
			if _, ok := expByID[expID]; ok {
				continue
			}

			set := component.ExporterCreateSettings{
				TelemetrySettings: settings,
				BuildInfo:         buildInfo,
			}
			set.TelemetrySettings.Logger = exporterLogger(settings.Logger, expID, dt)

			expCfg, existsCfg := cfg.Exporters[expID]
			if !existsCfg {
				return nil, fmt.Errorf("exporter %q is not configured", expID)
			}

			factory, existsFactory := factories[expID.Type()]
			if !existsFactory {
				return nil, fmt.Errorf("exporter factory not found for type: %s", expID.Type())
			}

			exp, err := buildExporter(ctx, factory, set, expCfg, pipelineID)
			if err != nil {
				return nil, err
			}

			expByID[expID] = exp
		}
	}
	return exps, nil
}

func buildExporter(
	ctx context.Context,
	factory component.ExporterFactory,
	set component.ExporterCreateSettings,
	cfg config.Exporter,
	pipelineID config.ComponentID,
) (component.Exporter, error) {
	var err error
	var exporter component.Exporter
	switch pipelineID.Type() {
	case config.TracesDataType:
		exporter, err = factory.CreateTracesExporter(ctx, set, cfg)

	case config.MetricsDataType:
		exporter, err = factory.CreateMetricsExporter(ctx, set, cfg)

	case config.LogsDataType:
		exporter, err = factory.CreateLogsExporter(ctx, set, cfg)

	default:
		// Could not create because this exporter does not support this data type.
		return nil, exporterTypeMismatchErr(cfg, pipelineID)
	}

	if err != nil {
		if errors.Is(err, component.ErrDataTypeIsNotSupported) {
			// Could not create because this exporter does not support this data type.
			return nil, exporterTypeMismatchErr(cfg, pipelineID)
		}
		return nil, fmt.Errorf("error creating %v exporter: %w", cfg.ID(), err)
	}

	set.Logger.Info("Exporter was built.")

	return exporter, nil
}

func exporterTypeMismatchErr(
	config config.Exporter,
	pipelineID config.ComponentID,
) error {
	return fmt.Errorf(
		"pipeline %q of data type %q has an exporter %v, which does not support that data type",
		pipelineID, pipelineID.Type(), config.ID(),
	)
}

func exporterLogger(logger *zap.Logger, id config.ComponentID, dt config.DataType) *zap.Logger {
	return logger.With(
		zap.String(components.ZapKindKey, components.ZapKindExporter),
		zap.String(components.ZapDataTypeKey, string(dt)),
		zap.String(components.ZapNameKey, id.String()))
}
