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
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/component/componenterror"
	"github.com/open-telemetry/opentelemetry-collector/config"
	"github.com/open-telemetry/opentelemetry-collector/config/configerror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
)

// builtExporter is an exporter that is built based on a config. It can have
// a trace and/or a metrics consumer and have a shutdown function.
type builtExporter struct {
	logger *zap.Logger

	// pipeline: exporter
	byPipeline map[string][]component.Exporter
}

// Start the exporter.
func (exp *builtExporter) Start(ctx context.Context, host component.Host) error {
	var errors []error
	for _, exps := range exp.byPipeline {
		for _, exp := range exps {
			if err := exp.Start(ctx, host); err != nil {
				errors = append(errors, err)
			}
		}
	}
	return componenterror.CombineErrors(errors)
}

// Shutdown the trace component and the metrics component of an exporter.
func (exp *builtExporter) Shutdown(ctx context.Context) error {
	var errors []error
	for _, exps := range exp.byPipeline {
		for _, exp := range exps {
			if err := exp.Shutdown(ctx); err != nil {
				errors = append(errors, err)
			}
		}
	}
	return componenterror.CombineErrors(errors)
}

func (exp *builtExporter) GetTraceExporter() component.TraceExporterBase {
	return nil
}

func (exp *builtExporter) GetMetricExporter() component.MetricsExporterBase {
	return nil
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

	if len(errs) != 0 {
		return componenterror.CombineErrors(errs)
	}
	return nil
}

// ToMapByDataType ...
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
	logger *zap.Logger
	config *configmodels.Config
	// factories map[configmodels.Type]component.ExporterFactoryBase
	factories config.Factories
}

// NewExportersBuilder creates a new ExportersBuilder. Call BuildExporters() on the returned value.
func NewExportersBuilder(
	logger *zap.Logger,
	config *configmodels.Config,
	factories config.Factories,
	// factories map[configmodels.Type]component.ExporterFactoryBase,
) *ExportersBuilder {
	return &ExportersBuilder{logger.With(zap.String(kindLogKey, kindLogExporter)), config, factories}
}

// Build builds exporters from config.
func (eb *ExportersBuilder) Build() (Exporters, error) {
	exporters := make(Exporters)

	// BuildExporters exporters based on configuration and required input data types.
	for _, cfg := range eb.config.Exporters {
		componentLogger := eb.logger.With(zap.String(typeLogKey, string(cfg.Type())), zap.String(nameLogKey, cfg.Name()))
		exp, err := eb.buildExporter(componentLogger, cfg)
		if err != nil {
			return nil, err
		}

		exporters[cfg] = exp
	}

	return exporters, nil
}

func (eb *ExportersBuilder) getPipelinesForExporter(exporterName string) []*configmodels.Pipeline {
	pipelines := map[*configmodels.Pipeline]bool{}
	// Iterate over pipelines.
	for _, pipeline := range eb.config.Service.Pipelines {
		// Iterate over all exporters for this pipeline.
		for _, expName := range pipeline.Exporters {
			if expName == exporterName {
				pipelines[pipeline] = true
			}
		}
	}

	results := []*configmodels.Pipeline{}
	for p := range pipelines {
		results = append(results, p)
	}
	return results
}

func (eb *ExportersBuilder) buildExporter(
	logger *zap.Logger,
	config configmodels.Exporter,
) (*builtExporter, error) {

	factory := eb.factories.Exporters[config.Type()]
	if factory == nil {
		return nil, fmt.Errorf("exporter factory not found for type: %s", config.Type())
	}

	exporter := &builtExporter{
		logger:     logger,
		byPipeline: map[string][]component.Exporter{},
	}

	ctx := context.Background()
	for _, p := range eb.getPipelinesForExporter(config.Name()) {
		pipelineFactory := eb.factories.Pipelines[p.Type()]

		exp, err := pipelineFactory.CreateExporter(ctx, factory, logger, config)
		if err != nil {
			if err == configerror.ErrDataTypeIsNotSupported {
				// Could not create because this exporter does not support this data type.
				return nil, typeMismatchErr(config, p)
			}
			return nil, fmt.Errorf("error creating %s exporter: %v", config.Name(), err)
		}
		if exp == nil {
			return nil, fmt.Errorf("factory for %q produced a nil exporter", config.Name())
		}
		exporters := exporter.byPipeline[p.Name]
		if exporters == nil {
			exporters = []component.Exporter{}
		}
		exporters = append(exporters, exp)
		exporter.byPipeline[p.Name] = exporters
	}

	eb.logger.Info("Exporter is enabled.", zap.String("exporter", config.Name()))
	return exporter, nil
}

func typeMismatchErr(
	config configmodels.Exporter,
	pipeline *configmodels.Pipeline,
) error {
	return fmt.Errorf("%s pipeline but has a %s which does not support %s",
		pipeline.Name, config.Name(), pipeline.Type(),
	)
}
