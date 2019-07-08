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

package jaegerexporter

import (
	"fmt"

	"go.uber.org/zap"

	"contrib.go.opencensus.io/exporter/jaeger"
	"github.com/open-telemetry/opentelemetry-service/config/configerror"
	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/exporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/exporterwrapper"
)

var _ = exporter.RegisterFactory(&factory{})

const (
	// The value of "type" key in configuration.
	typeStr = "jaeger"
)

// factory is the factory for Jaeger exporter.
type factory struct {
}

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
func (f *factory) CreateTraceExporter(logger *zap.Logger, config configmodels.Exporter) (consumer.TraceConsumer, exporter.StopFunc, error) {
	jc := config.(*ConfigV2)

	if jc.CollectorEndpoint == "" {
		return nil, nil, &jTraceExporterError{
			code: errCollectorEndpointRequired,
			msg:  "Jaeger exporter config requires an endpoint",
		}
	}

	jOptions := jaeger.Options{}
	jOptions.CollectorEndpoint = jc.CollectorEndpoint

	if jc.Username == "" {
		return nil, nil, &jTraceExporterError{
			code: errUsernameRequired,
			msg:  "Jaeger exporter config requires a username",
		}
	}
	jOptions.Username = jc.Username

	if jc.Password == "" {
		return nil, nil, &jTraceExporterError{
			code: errPasswordRequired,
			msg:  "Jaeger exporter config requires a Password",
		}
	}
	jOptions.Password = jc.Password

	if jc.Password == "" {
		return nil, nil, &jTraceExporterError{
			code: errServiceNameRequired,
			msg:  "Jaeger exporter config requires a ServiceName",
		}
	}
	jOptions.Process = jaeger.Process{
		ServiceName: jc.ServiceName,
	}

	exporter, serr := jaeger.NewExporter(jOptions)
	if serr != nil {
		return nil, nil, fmt.Errorf("cannot create Jaeger trace exporter: %v", serr)
	}
	// wrap exporter to return exporter.stop()
	je := jaegerExporter{exporter}

	jexp, err := exporterwrapper.NewExporterWrapper("jaeger", "ocservice.exporter.Jaeger.ConsumeTraceData", exporter)
	if err != nil {
		return nil, nil, err
	}

	return jexp, je.stop, nil
}

// CreateMetricsExporter creates a metrics exporter based on this config.
func (f *factory) CreateMetricsExporter(logger *zap.Logger, cfg configmodels.Exporter) (consumer.MetricsConsumer, exporter.StopFunc, error) {
	return nil, nil, configerror.ErrDataTypeIsNotSupported
}

func (je *jaegerExporter) stop() error {
	// waiting for exported trace spans to be uploaded before stopping.
	je.exporter.Flush()
	return nil
}
