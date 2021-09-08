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
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumererror"
)

// builtExporter is an exporter that is built based on a config. It can have
// a trace and/or a metrics consumer and have a shutdown function.
type builtExporter struct {
	logger        *zap.Logger
	expByDataType map[config.DataType]component.Exporter
}

// Start the exporter.
func (bexp *builtExporter) Start(ctx context.Context, host component.Host) error {
	var errors []error
	for _, exporter := range bexp.expByDataType {
		err := exporter.Start(ctx, host)
		if err != nil {
			errors = append(errors, err)
		}
	}

	return consumererror.Combine(errors)
}

// Shutdown the trace component and the metrics component of an exporter.
func (bexp *builtExporter) Shutdown(ctx context.Context) error {
	var errors []error
	for _, exporter := range bexp.expByDataType {
		err := exporter.Shutdown(ctx)
		if err != nil {
			errors = append(errors, err)
		}
	}

	return consumererror.Combine(errors)
}

func (bexp *builtExporter) getTracesExporter() component.TracesExporter {
	exp := bexp.expByDataType[config.TracesDataType]
	if exp == nil {
		return nil
	}
	return exp.(component.TracesExporter)
}

func (bexp *builtExporter) getMetricExporter() component.MetricsExporter {
	exp := bexp.expByDataType[config.MetricsDataType]
	if exp == nil {
		return nil
	}
	return exp.(component.MetricsExporter)
}

func (bexp *builtExporter) getLogExporter() component.LogsExporter {
	exp := bexp.expByDataType[config.LogsDataType]
	if exp == nil {
		return nil
	}
	return exp.(component.LogsExporter)
}

// Exporters is a map of exporters created from exporter configs.
type Exporters map[config.ComponentID]*builtExporter

// StartAll starts all exporters.
func (exps Exporters) StartAll(ctx context.Context, host component.Host) error {
	for _, exp := range exps {
		exp.logger.Info("Exporter is starting...")

		if err := exp.Start(ctx, newHostWrapper(host, exp.logger)); err != nil {
			return err
		}
		exp.logger.Info("Exporter started.")
	}
	return nil
}

// ShutdownAll stops all exporters.
func (exps Exporters) ShutdownAll(ctx context.Context) error {
	var errs []error
	for _, exp := range exps {
		err := exp.Shutdown(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return consumererror.Combine(errs)
}

func (exps Exporters) ToMapByDataType() map[config.DataType]map[config.ComponentID]component.Exporter {

	exportersMap := make(map[config.DataType]map[config.ComponentID]component.Exporter)

	exportersMap[config.TracesDataType] = make(map[config.ComponentID]component.Exporter, len(exps))
	exportersMap[config.MetricsDataType] = make(map[config.ComponentID]component.Exporter, len(exps))
	exportersMap[config.LogsDataType] = make(map[config.ComponentID]component.Exporter, len(exps))

	for expID, bexp := range exps {
		for t, exp := range bexp.expByDataType {
			exportersMap[t][expID] = exp
		}
	}

	return exportersMap
}

type dataTypeRequirement struct {
	// Pipeline that requires the data type.
	requiredBy *config.Pipeline
}

// Map of data type requirements.
type dataTypeRequirements map[config.DataType]dataTypeRequirement

// Data type requirements for all exporters.
type exportersRequiredDataTypes map[config.ComponentID]dataTypeRequirements

// BuildExporters builds Exporters from config.
func BuildExporters(
	logger *zap.Logger,
	tracerProvider trace.TracerProvider,
	buildInfo component.BuildInfo,
	cfg *config.Config,
	factories map[config.Type]component.ExporterFactory,
) (Exporters, error) {
	logger = logger.With(zap.String(zapKindKey, zapKindLogExporter))

	// We need to calculate required input data types for each exporter so that we know
	// which data type must be started for each exporter.
	exporterInputDataTypes := calcExportersRequiredDataTypes(cfg)

	exporters := make(Exporters)

	// Build exporters exporters based on configuration and required input data types.
	for expID, expCfg := range cfg.Exporters {
		set := component.ExporterCreateSettings{
			TelemetryCreateSettings: component.TelemetryCreateSettings{
				Logger:         logger.With(zap.String(zapNameKey, expID.String())),
				TracerProvider: tracerProvider,
			},
			BuildInfo: buildInfo,
		}

		factory, exists := factories[expID.Type()]
		if !exists || factory == nil {
			return nil, fmt.Errorf("exporter factory not found for type: %s", expID.Type())
		}

		exp, err := buildExporter(context.Background(), factory, set, expCfg, exporterInputDataTypes[expID])
		if err != nil {
			return nil, err
		}

		exporters[expID] = exp
	}

	return exporters, nil
}

func calcExportersRequiredDataTypes(cfg *config.Config) exportersRequiredDataTypes {
	// Go over all pipelines. The data type of the pipeline defines what data type
	// each exporter is expected to receive. Collect all required types for each
	// exporter.
	//
	// We also remember the last pipeline that requested the particular data type.
	// This is only needed for logging purposes in error cases when we need to
	// print that a particular exporter does not support the data type required for
	// a particular pipeline.

	result := make(exportersRequiredDataTypes)

	// Iterate over pipelines.
	for _, pipeline := range cfg.Service.Pipelines {
		// Iterate over all exporters for this pipeline.
		for _, expID := range pipeline.Exporters {
			// Create the data type requirement for the expCfg if it does not exist.
			if _, ok := result[expID]; !ok {
				result[expID] = make(dataTypeRequirements)
			}

			// Remember that this data type is required for the expCfg and also which
			// pipeline the requirement is coming from.
			result[expID][pipeline.InputType] = dataTypeRequirement{pipeline}
		}
	}
	return result
}

func buildExporter(
	ctx context.Context,
	factory component.ExporterFactory,
	set component.ExporterCreateSettings,
	cfg config.Exporter,
	inputDataTypes dataTypeRequirements,
) (*builtExporter, error) {
	exporter := &builtExporter{
		logger:        set.Logger,
		expByDataType: make(map[config.DataType]component.Exporter, 3),
	}

	if inputDataTypes == nil {
		set.Logger.Info("Ignoring exporter as it is not used by any pipeline")
		return exporter, nil
	}

	var err error
	var createdExporter component.Exporter
	for dataType, requirement := range inputDataTypes {
		switch dataType {
		case config.TracesDataType:
			createdExporter, err = factory.CreateTracesExporter(ctx, set, cfg)

		case config.MetricsDataType:
			createdExporter, err = factory.CreateMetricsExporter(ctx, set, cfg)

		case config.LogsDataType:
			createdExporter, err = factory.CreateLogsExporter(ctx, set, cfg)

		default:
			// Could not create because this exporter does not support this data type.
			return nil, exporterTypeMismatchErr(cfg, requirement.requiredBy, dataType)
		}

		if err != nil {
			if err == componenterror.ErrDataTypeIsNotSupported {
				// Could not create because this exporter does not support this data type.
				return nil, exporterTypeMismatchErr(cfg, requirement.requiredBy, dataType)
			}
			return nil, fmt.Errorf("error creating %v exporter: %v", cfg.ID(), err)
		}

		// Check if the factory really created the exporter.
		if createdExporter == nil {
			return nil, fmt.Errorf("factory for %v produced a nil exporter", cfg.ID())
		}

		exporter.expByDataType[dataType] = createdExporter
	}

	set.Logger.Info("Exporter was built.")

	return exporter, nil
}

func exporterTypeMismatchErr(
	config config.Exporter,
	requiredByPipeline *config.Pipeline,
	dataType config.DataType,
) error {
	return fmt.Errorf(
		"pipeline %q of data type %q has an exporter %v, which does not support that data type",
		requiredByPipeline.Name, dataType, config.ID(),
	)
}
