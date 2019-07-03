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
	"testing"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/models"
)

type ExampleExporterFactory struct {
}

// Type gets the type of the Exporter config created by this factory.
func (f *ExampleExporterFactory) Type() string {
	return "exampleexporter"
}

// CreateDefaultConfig creates the default configuration for the Exporter.
func (f *ExampleExporterFactory) CreateDefaultConfig() models.Exporter {
	return nil
}

// CreateTraceExporter creates a trace exporter based on this config.
func (f *ExampleExporterFactory) CreateTraceExporter(logger *zap.Logger, cfg models.Exporter) (consumer.TraceConsumer, StopFunc, error) {
	return nil, nil, nil
}

// CreateMetricsExporter creates a metrics exporter based on this config.
func (f *ExampleExporterFactory) CreateMetricsExporter(logger *zap.Logger, cfg models.Exporter) (consumer.MetricsConsumer, StopFunc, error) {
	return nil, nil, nil
}

func TestRegisterExporterFactory(t *testing.T) {
	f := ExampleExporterFactory{}
	err := RegisterExporterFactory(&f)
	if err != nil {
		t.Fatalf("cannot register factory")
	}

	if &f != GetExporterFactory(f.Type()) {
		t.Fatalf("cannot find factory")
	}

	// Verify that attempt to register a factory with duplicate name panics
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()

		err = RegisterExporterFactory(&f)
	}()

	if !panicked {
		t.Fatalf("must panic on double registration")
	}
}
