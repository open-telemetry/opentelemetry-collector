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
	"fmt"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/models"
)

// StopFunc is a function that can be called to stop an exporter that
// was created previously.
type StopFunc func() error

// Factory interface for exporters.
type Factory interface {
	// Type gets the type of the Exporter created by this factory.
	Type() string

	// CreateDefaultConfig creates the default configuration for the Exporter.
	CreateDefaultConfig() models.Exporter

	// CreateTraceExporter creates a trace exporter based on this config.
	CreateTraceExporter(logger *zap.Logger, cfg models.Exporter) (consumer.TraceConsumer, StopFunc, error)

	// CreateMetricsExporter creates a metrics exporter based on this config.
	CreateMetricsExporter(logger *zap.Logger, cfg models.Exporter) (consumer.MetricsConsumer, StopFunc, error)
}

// List of registered exporter factories.
var exporterFactories = make(map[string]Factory)

// RegisterFactory registers a exporter factory.
func RegisterFactory(factory Factory) error {
	if exporterFactories[factory.Type()] != nil {
		panic(fmt.Sprintf("duplicate exporter factory %q", factory.Type()))
	}

	exporterFactories[factory.Type()] = factory
	return nil
}

// GetFactory gets a exporter factory by type string.
func GetFactory(typeStr string) Factory {
	return exporterFactories[typeStr]
}
