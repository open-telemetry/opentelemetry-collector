// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package builder

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/config/configerror"
	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/exporter"
	"github.com/open-telemetry/opentelemetry-service/internal"
)

// builtExporter is an exporter that is built based on a config. It can have
// a trace and/or a metrics consumer and have a stop function.
type builtExporter struct {
	tc   consumer.TraceConsumer
	mc   consumer.MetricsConsumer
	stop func() error
}

// Stop the exporter.
func (exp *builtExporter) Stop() error {
	return exp.stop()
}

// Exporters is a map of exporters created from exporter configs.
type Exporters map[configmodels.Exporter]*builtExporter

// StopAll stops all exporters.
func (exps Exporters) StopAll() {
	for _, exp := range exps {
		exp.Stop()
	}
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
	config    *configmodels.Config
	factories map[string]exporter.Factory
}

// NewExportersBuilder creates a new ExportersBuilder. Call Build() on the returned value.
func NewExportersBuilder(
	logger *zap.Logger,
	config *configmodels.Config,
	factories map[string]exporter.Factory,
) *ExportersBuilder {
	return &ExportersBuilder{logger, config, factories}
}

// Build exporters from config.
func (eb *ExportersBuilder) Build() (Exporters, error) {
	exporters := make(Exporters)

	// We need to calculate required input data types for each exporter so that we know
	// which data type must be started for each exporter.
	exporterInputDataTypes := eb.calcExportersRequiredDataTypes()

	// Build exporters based on configuration and required input data types.
	for _, cfg := range eb.config.Exporters {
		exp, err := eb.buildExporter(cfg, exporterInputDataTypes)
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
	for _, pipeline := range eb.config.Pipelines {
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

// combineStopFunc combines 2 functions and returns one function
// that can be called and which in turn will call both functions
// and then combine any errors that the 2 functions return.
// Safe to use if any of the 2 functions are nil. If both functions
// are nil then returns nil.
func combineStopFunc(f1, f2 exporter.StopFunc) exporter.StopFunc {
	if f1 == nil {
		return f2
	}
	if f2 == nil {
		return f1
	}

	return func() error {
		var errs []error
		if err := f1(); err != nil {
			errs = append(errs, err)
		}
		if err := f2(); err != nil {
			errs = append(errs, err)
		}
		return internal.CombineErrors(errs)
	}
}

func (eb *ExportersBuilder) buildExporter(
	config configmodels.Exporter,
	exportersInputDataTypes exportersRequiredDataTypes,
) (*builtExporter, error) {

	factory := eb.factories[config.Type()]
	if factory == nil {
		return nil, fmt.Errorf("exporter factory not found for type: %s", config.Type())
	}

	exporter := &builtExporter{}

	inputDataTypes := exportersInputDataTypes[config]
	if inputDataTypes == nil {
		// No data types where requested for this exporter. This can only happen
		// if there are no pipelines associated with the exporter.
		eb.logger.Warn("Exporter " + config.Name() +
			" is not associated with any pipeline and will not export data.")
		return exporter, nil
	}

	if requirement, ok := inputDataTypes[configmodels.TracesDataType]; ok {
		// Traces data type is required. Create a trace exporter based on config.
		tc, stopFunc, err := factory.CreateTraceExporter(eb.logger, config)
		if err != nil {
			if err == configerror.ErrDataTypeIsNotSupported {
				// Could not create because this exporter does not support this data type.
				return nil, typeMismatchErr(config, requirement.requiredBy, configmodels.TracesDataType)
			}
			return nil, fmt.Errorf("error creating %s exporter: %v", config.Name(), err)
		}

		exporter.tc = tc
		exporter.stop = stopFunc
	}

	if requirement, ok := inputDataTypes[configmodels.MetricsDataType]; ok {
		// Metrics data type is required. Create a trace exporter based on config.
		mc, stopFunc, err := factory.CreateMetricsExporter(eb.logger, config)
		if err != nil {
			if err == configerror.ErrDataTypeIsNotSupported {
				// Could not create because this exporter does not support this data type.
				return nil, typeMismatchErr(config, requirement.requiredBy, configmodels.MetricsDataType)
			}
			return nil, fmt.Errorf("error creating %s exporter: %v", config.Name(), err)
		}

		exporter.mc = mc
		exporter.stop = combineStopFunc(exporter.stop, stopFunc)
	}

	eb.logger.Info("Exporter is enabled.", zap.String("exporter", config.Name()))

	return exporter, nil
}

func typeMismatchErr(
	config configmodels.Exporter,
	requiredByPipeline *configmodels.Pipeline,
	dataType configmodels.DataType,
) error {
	return fmt.Errorf("%s is a %s pipeline but has a %s which does not support %s",
		requiredByPipeline.Name, dataType.GetString(),
		config.Name(), dataType.GetString(),
	)
}
