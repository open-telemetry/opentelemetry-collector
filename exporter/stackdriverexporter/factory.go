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

package stackdriverexporter

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"github.com/open-telemetry/opentelemetry-service/config/configerror"
	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/exporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/exporterwrapper"
)

var _ = exporter.RegisterFactory(&factory{})

const (
	// The value of "type" key in configuration.
	typeStr = "stackdriver"
)

// factory is the factory for Prometheus exporter.
type factory struct {
}

var sde *stackdriver.Exporter

// Type gets the type of the Exporter config created by this factory.
func (f *factory) Type() string {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for exporter.
func (f *factory) CreateDefaultConfig() configmodels.Exporter {
	return &ConfigV2{
		ExporterSettings: configmodels.ExporterSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
	}
}

// CreateTraceExporter creates a trace exporter based on this config.
func (f *factory) CreateTraceExporter(logger *zap.Logger, cfg configmodels.Exporter) (consumer.TraceConsumer, exporter.StopFunc, error) {
	scfg := cfg.(*ConfigV2)

	if !scfg.EnableTracing && !scfg.EnableMetrics {
		return nil, nil, nil
	}

	// TODO:  For each ProjectID, create a different exporter
	// or at least a unique Stackdriver client per ProjectID.

	sde, err := stackdriver.NewExporter(stackdriver.Options{
		// If the project ID is an empty string, it will be set by default based on
		// the project this is running on in GCP.
		ProjectID: scfg.ProjectID,

		MetricPrefix: scfg.MetricPrefix,

		// Stackdriver Metrics mandates a minimum of 60 seconds for
		// reporting metrics. We have to enforce this as per the advisory
		// at https://cloud.google.com/monitoring/custom-metrics/creating-metrics#writing-ts
		// which says:
		//
		// "If you want to write more than one point to the same time series, then use a separate call
		//  to the timeSeries.create method for each point. Don't make the calls faster than one time per
		//  minute. If you are adding data points to different time series, then there is no rate limitation."
		BundleDelayThreshold: 61 * time.Second,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("Cannot configure Stackdriver Trace exporter: %v", err)
	}

	sexp, err := exporterwrapper.NewExporterWrapper("stackdriver_trace", "ocservice.exporter.Stackdriver.ConsumeTraceData", sde)
	if err != nil {
		return nil, nil, err
	}

	return sexp, stopFunc, nil
}

// CreateMetricsExporter creates a metrics exporter based on this config.
func (f *factory) CreateMetricsExporter(logger *zap.Logger, cfg configmodels.Exporter) (consumer.MetricsConsumer, exporter.StopFunc, error) {
	return nil, nil, configerror.ErrDataTypeIsNotSupported
}

func stopFunc() error {
	sde.Flush()
	return nil
}
