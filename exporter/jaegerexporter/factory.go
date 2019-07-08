package jaegerexporter

import (
	"fmt"

	"contrib.go.opencensus.io/exporter/jaeger"
	"github.com/open-telemetry/opentelemetry-service/configv2/configerror"
	"github.com/open-telemetry/opentelemetry-service/configv2/configmodels"
	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/exporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/exporterwrapper"
	"go.uber.org/zap"
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
			msg:  "Jaeger exporter config requires an Endpoint",
		}
	}

	jOptions := jaeger.Options{}
	jOptions.CollectorEndpoint = jc.CollectorEndpoint

	if jc.Username == "" {
		return nil, nil, &jTraceExporterError{
			code: errUsernameRequired,
			msg:  "Jaeger exporter config requires a Username",
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
		return nil, nil, fmt.Errorf("cannot configure jaeger Trace exporter: %v", serr)
	}

	jexp, err := exporterwrapper.NewExporterWrapper("jaeger", "ocservice.exporter.Jaeger.ConsumeTraceData", exporter)
	if err != nil {
		return nil, nil, err
	}

	return jexp, noopStopFunc, nil
}

// CreateMetricsExporter creates a metrics exporter based on this config.
func (f *factory) CreateMetricsExporter(logger *zap.Logger, cfg configmodels.Exporter) (consumer.MetricsConsumer, exporter.StopFunc, error) {
	return nil, nil, configerror.ErrDataTypeIsNotSupported
}

func noopStopFunc() error {
	return nil
}
