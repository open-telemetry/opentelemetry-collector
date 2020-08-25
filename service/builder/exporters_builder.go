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
	logger *zap.Logger
	te     component.TraceExporter
	me     component.MetricsExporter
	le     component.LogsExporter
}

// Start the exporter.
func (exp *builtExporter) Start(ctx context.Context, host component.Host) error {
	var errors []error
	if exp.te != nil {
		err := exp.te.Start(ctx, host)
		if err != nil {
			errors = append(errors, err)
		}
	}

	if exp.me != nil {
		err := exp.me.Start(ctx, host)
		if err != nil {
			errors = append(errors, err)
		}
	}

	return componenterror.CombineErrors(errors)
}

// Shutdown the trace component and the metrics component of an exporter.
func (exp *builtExporter) Shutdown(ctx context.Context) error {
	var errors []error
	if exp.te != nil {
		if err := exp.te.Shutdown(ctx); err != nil {
			errors = append(errors, err)
		}
	}

	if exp.me != nil {
		if err := exp.me.Shutdown(ctx); err != nil {
			errors = append(errors, err)
		}
	}

	return componenterror.CombineErrors(errors)
}

func (exp *builtExporter) GetTraceExporter() component.TraceExporter {
	return exp.te
}

func (exp *builtExporter) GetMetricExporter() component.MetricsExporter {
	return exp.me
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

	for cfg, exp := range exps {
		te := exp.GetTraceExporter()
		if te != nil {
			exportersMap[configmodels.TracesDataType][cfg] = te
		}
		me := exp.GetMetricExporter()
		if me != nil {
			exportersMap[configmodels.MetricsDataType][cfg] = me
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
		exp, err := eb.buildExporter(componentLogger, eb.appInfo, cfg, exporterInputDataTypes)
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
		logger: logger,
	}

	inputDataTypes := exportersInputDataTypes[config]
	if inputDataTypes == nil {
		logger.Info("Ignoring exporter as it is not used by any pipeline")
		return exporter, nil
	}

	for dataType, requirement := range inputDataTypes {

		switch dataType {
		case configmodels.TracesDataType:
			// Traces data type is required. Create a trace exporter based on config.
			te, err := createTraceExporter(factory, logger, appInfo, config)
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

			exporter.te = te

		case configmodels.MetricsDataType:
			// Metrics data type is required. Create a trace exporter based on config.
			me, err := createMetricsExporter(factory, logger, appInfo, config)
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

			exporter.me = me

		case configmodels.LogsDataType:
			le, err := createLogsExporter(factory, logger, appInfo, config)
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

			exporter.le = le

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

// createTraceProcessor creates a trace exporter using given factory.
func createTraceExporter(
	factory component.ExporterFactory,
	logger *zap.Logger,
	appInfo component.ApplicationStartInfo,
	cfg configmodels.Exporter,
) (component.TraceExporter, error) {
	creationParams := component.ExporterCreateParams{
		Logger:               logger,
		ApplicationStartInfo: appInfo,
	}
	return factory.CreateTraceExporter(context.Background(), creationParams, cfg)
}

// createMetricsExporter creates a metrics exporter using given factory.
func createMetricsExporter(
	factory component.ExporterFactory,
	logger *zap.Logger,
	appInfo component.ApplicationStartInfo,
	cfg configmodels.Exporter,
) (component.MetricsExporter, error) {
	creationParams := component.ExporterCreateParams{
		Logger:               logger,
		ApplicationStartInfo: appInfo,
	}
	return factory.CreateMetricsExporter(context.Background(), creationParams, cfg)
}

// createLogsExporter creates a log exporter using given factory.
func createLogsExporter(
	factory component.ExporterFactory,
	logger *zap.Logger,
	appInfo component.ApplicationStartInfo,
	cfg configmodels.Exporter,
) (component.LogsExporter, error) {
	creationParams := component.ExporterCreateParams{
		Logger:               logger,
		ApplicationStartInfo: appInfo,
	}
	return factory.CreateLogsExporter(context.Background(), creationParams, cfg)
}
