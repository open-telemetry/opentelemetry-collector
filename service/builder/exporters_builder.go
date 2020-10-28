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
	"go.opentelemetry.io/collector/config/configerror"
	"go.opentelemetry.io/collector/config/configmodels"
)

// builtExporter is an exporter that is built based on a config. It can have
// a trace and/or a metrics consumer and have a shutdown function.
type builtExporter struct {
	logger        *zap.Logger
	expByDataType map[configmodels.DataType]component.Exporter
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

	return componenterror.CombineErrors(errors)
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

	return componenterror.CombineErrors(errors)
}

func (bexp *builtExporter) getTraceExporter() component.TracesExporter {
	exp := bexp.expByDataType[configmodels.TracesDataType]
	if exp == nil {
		return nil
	}
	return exp.(component.TracesExporter)
}

func (bexp *builtExporter) getMetricExporter() component.MetricsExporter {
	exp := bexp.expByDataType[configmodels.MetricsDataType]
	if exp == nil {
		return nil
	}
	return exp.(component.MetricsExporter)
}

func (bexp *builtExporter) getLogExporter() component.LogsExporter {
	exp := bexp.expByDataType[configmodels.LogsDataType]
	if exp == nil {
		return nil
	}
	return exp.(component.LogsExporter)
}

// Exporters is a map of exporters created from exporter configs.
type Exporters map[configmodels.Exporter]*builtExporter

// StartAll starts all exporters.
func (exps Exporters) StartAll(ctx context.Context, host component.Host) error {
	for _, exp := range exps {
		exp.logger.Info("Exporter is starting...")

		if err := exp.Start(ctx, host); err != nil {
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

	return componenterror.CombineErrors(errs)
}

func (exps Exporters) ToMapByDataType() map[configmodels.DataType]map[configmodels.Exporter]component.Exporter {

	exportersMap := make(map[configmodels.DataType]map[configmodels.Exporter]component.Exporter)

	exportersMap[configmodels.TracesDataType] = make(map[configmodels.Exporter]component.Exporter, len(exps))
	exportersMap[configmodels.MetricsDataType] = make(map[configmodels.Exporter]component.Exporter, len(exps))
	exportersMap[configmodels.LogsDataType] = make(map[configmodels.Exporter]component.Exporter, len(exps))

	for cfg, bexp := range exps {
		for t, exp := range bexp.expByDataType {
			exportersMap[t][cfg] = exp
		}
	}

	return exportersMap
}

type dataTypeRequirement struct {
	// Pipeline that requires the data type.
	requiredBy *configmodels.Pipeline
}

// Map of data type requirements.
type dataTypeRequirements map[configmodels.DataType]dataTypeRequirement

// Data type requirements for all exporters.
type exportersRequiredDataTypes map[configmodels.Exporter]dataTypeRequirements

// ExportersBuilder builds exporters from config.
type ExportersBuilder struct {
	logger    *zap.Logger
	appInfo   component.ApplicationStartInfo
	config    *configmodels.Config
	factories map[configmodels.Type]component.ExporterFactory
}

// NewExportersBuilder creates a new ExportersBuilder. Call BuildExporters() on the returned value.
func NewExportersBuilder(
	logger *zap.Logger,
	appInfo component.ApplicationStartInfo,
	config *configmodels.Config,
	factories map[configmodels.Type]component.ExporterFactory,
) *ExportersBuilder {
	return &ExportersBuilder{logger.With(zap.String(kindLogKey, kindLogsExporter)), appInfo, config, factories}
}

// BuildExporters exporters from config.
func (eb *ExportersBuilder) Build() (Exporters, error) {
	exporters := make(Exporters)

	// We need to calculate required input data types for each exporter so that we know
	// which data type must be started for each exporter.
	exporterInputDataTypes := eb.calcExportersRequiredDataTypes()

	// BuildExporters exporters based on configuration and required input data types.
	for _, cfg := range eb.config.Exporters {
		componentLogger := eb.logger.With(zap.String(typeLogKey, string(cfg.Type())), zap.String(nameLogKey, cfg.Name()))
		exp, err := eb.buildExporter(context.Background(), componentLogger, eb.appInfo, cfg, exporterInputDataTypes)
		if err != nil {
			return nil, err
		}

		exporters[cfg] = exp
	}

	return exporters, nil
}

func (eb *ExportersBuilder) calcExportersRequiredDataTypes() exportersRequiredDataTypes {

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
	for _, pipeline := range eb.config.Service.Pipelines {
		// Iterate over all exporters for this pipeline.
		for _, expName := range pipeline.Exporters {
			// Find the exporter config by name.
			exporter := eb.config.Exporters[expName]

			// Create the data type requirement for the exporter if it does not exist.
			if result[exporter] == nil {
				result[exporter] = make(dataTypeRequirements)
			}

			// Remember that this data type is required for the exporter and also which
			// pipeline the requirement is coming from.
			result[exporter][pipeline.InputType] = dataTypeRequirement{pipeline}
		}
	}
	return result
}

func (eb *ExportersBuilder) buildExporter(
	ctx context.Context,
	logger *zap.Logger,
	appInfo component.ApplicationStartInfo,
	config configmodels.Exporter,
	exportersInputDataTypes exportersRequiredDataTypes,
) (*builtExporter, error) {
	factory := eb.factories[config.Type()]
	if factory == nil {
		return nil, fmt.Errorf("exporter factory not found for type: %s", config.Type())
	}

	exporter := &builtExporter{
		logger:        logger,
		expByDataType: make(map[configmodels.DataType]component.Exporter, 3),
	}

	inputDataTypes := exportersInputDataTypes[config]
	if inputDataTypes == nil {
		eb.logger.Info("Ignoring exporter as it is not used by any pipeline")
		return exporter, nil
	}

	creationParams := component.ExporterCreateParams{
		Logger:               logger,
		ApplicationStartInfo: appInfo,
	}

	for dataType, requirement := range inputDataTypes {
		switch dataType {
		case configmodels.TracesDataType:
			// Traces data type is required. Create a trace exporter based on config.
			te, err := factory.CreateTracesExporter(ctx, creationParams, config)
			if err != nil {
				if err == configerror.ErrDataTypeIsNotSupported {
					// Could not create because this exporter does not support this data type.
					return nil, exporterTypeMismatchErr(config, requirement.requiredBy, dataType)
				}
				return nil, fmt.Errorf("error creating %s exporter: %v", config.Name(), err)
			}

			// Check if the factory really created the exporter.
			if te == nil {
				return nil, fmt.Errorf("factory for %q produced a nil exporter", config.Name())
			}

			exporter.expByDataType[configmodels.TracesDataType] = te

		case configmodels.MetricsDataType:
			// Metrics data type is required. Create a trace exporter based on config.
			me, err := factory.CreateMetricsExporter(ctx, creationParams, config)
			if err != nil {
				if err == configerror.ErrDataTypeIsNotSupported {
					// Could not create because this exporter does not support this data type.
					return nil, exporterTypeMismatchErr(config, requirement.requiredBy, dataType)
				}
				return nil, fmt.Errorf("error creating %s exporter: %v", config.Name(), err)
			}

			// The factories can be implemented by third parties, check if they really
			// created the exporter.
			if me == nil {
				return nil, fmt.Errorf("factory for %q produced a nil exporter", config.Name())
			}

			exporter.expByDataType[configmodels.MetricsDataType] = me

		case configmodels.LogsDataType:
			le, err := factory.CreateLogsExporter(ctx, creationParams, config)
			if err != nil {
				if err == configerror.ErrDataTypeIsNotSupported {
					// Could not create because this exporter does not support this data type.
					return nil, exporterTypeMismatchErr(config, requirement.requiredBy, dataType)
				}
				return nil, fmt.Errorf("error creating %s exporter: %v", config.Name(), err)
			}

			// Check if the factory really created the exporter.
			if le == nil {
				return nil, fmt.Errorf("factory for %q produced a nil exporter", config.Name())
			}

			exporter.expByDataType[configmodels.LogsDataType] = le

		default:
			// Could not create because this exporter does not support this data type.
			return nil, exporterTypeMismatchErr(config, requirement.requiredBy, dataType)
		}
	}

	eb.logger.Info("Exporter is enabled.", zap.String("exporter", config.Name()))

	return exporter, nil
}

func exporterTypeMismatchErr(
	config configmodels.Exporter,
	requiredByPipeline *configmodels.Pipeline,
	dataType configmodels.DataType,
) error {
	return fmt.Errorf("pipeline %q of data type %q has an exporter %q, which does not support that data type",
		requiredByPipeline.Name, dataType,
		config.Name(),
	)
}
