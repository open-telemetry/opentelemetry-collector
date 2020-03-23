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

package exporter

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
)

// BaseFactory defines the common functions for all exporter factories.
type BaseFactory interface {
	// Type gets the type of the Exporter created by this factory.
	Type() string

	// CreateDefaultConfig creates the default configuration for the Exporter.
	// This method can be called multiple times depending on the pipeline
	// configuration and should not cause side-effects that prevent the creation
	// of multiple instances of the Exporter.
	// The object returned by this method needs to pass the checks implemented by
	// 'configcheck.ValidateConfig'. It is recommended to have such check in the
	// tests of any implementation of the Factory interface.
	CreateDefaultConfig() configmodels.Exporter
}

// Factory can create TraceExporter and MetricsExporter.
type Factory interface {
	BaseFactory

	// CreateTraceExporter creates a trace exporter based on this config.
	CreateTraceExporter(logger *zap.Logger, cfg configmodels.Exporter) (TraceExporter, error)

	// CreateMetricsExporter creates a metrics exporter based on this config.
	CreateMetricsExporter(logger *zap.Logger, cfg configmodels.Exporter) (MetricsExporter, error)
}

// CreationParams is passed to Create* functions in FactoryV2.
type CreationParams struct {
	// Logger that the factory can use during creation and can pass to the created
	// component to be used later as well.
	Logger *zap.Logger
}

// FactoryV2 can create TraceExporterV2 and MetricsExporterV2. This is the
// new factory type that can create new style exporters.
type FactoryV2 interface {
	BaseFactory

	// CreateTraceExporter creates a trace exporter based on this config.
	// If the exporter type does not support tracing or if the config is not valid
	// error will be returned instead.
	CreateTraceExporter(ctx context.Context, params CreationParams, cfg configmodels.Exporter) (TraceExporterV2, error)

	// CreateMetricsExporter creates a metrics exporter based on this config.
	// If the exporter type does not support metrics or if the config is not valid
	// error will be returned instead.
	CreateMetricsExporter(ctx context.Context, params CreationParams, cfg configmodels.Exporter) (MetricsExporterV2, error)
}

// Build takes a list of exporter factories and returns a map of type map[string]Factory
// with factory type as keys. It returns a non-nil error when more than one factories
// have the same type.
func Build(factories ...Factory) (map[string]Factory, error) {
	fMap := map[string]Factory{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate exporter factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}
