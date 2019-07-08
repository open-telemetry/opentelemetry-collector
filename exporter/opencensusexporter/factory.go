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

package opencensusexporter

import (
	"crypto/x509"
	"fmt"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	"github.com/open-telemetry/opentelemetry-service/config/configerror"
	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/exporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/exporterhelper"
	"github.com/open-telemetry/opentelemetry-service/internal/compression"
	compressiongrpc "github.com/open-telemetry/opentelemetry-service/internal/compression/grpc"
)

var _ = exporter.RegisterFactory(&factory{})

const (
	// The value of "type" key in configuration.
	typeStr = "opencensus"
)

// factory is the factory for OpenCensus exporter.
type factory struct {
}

// Type gets the type of the Exporter config created by this factory.
func (f *factory) Type() string {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for exporter.
func (f *factory) CreateDefaultConfig() configmodels.Exporter {
	return &Config{
		ExporterSettings: configmodels.ExporterSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		Headers: map[string]string{},
	}
}

// CreateTraceExporter creates a trace exporter based on this config.
func (f *factory) CreateTraceExporter(logger *zap.Logger, config configmodels.Exporter) (consumer.TraceConsumer, exporter.StopFunc, error) {
	ocac := config.(*Config)

	if ocac.Endpoint == "" {
		return nil, nil, &ocTraceExporterError{
			code: errEndpointRequired,
			msg:  "OpenCensus exporter config requires an Endpoint",
		}
	}

	opts := []ocagent.ExporterOption{ocagent.WithAddress(ocac.Endpoint)}
	if ocac.Compression != "" {
		if compressionKey := compressiongrpc.GetGRPCCompressionKey(ocac.Compression); compressionKey != compression.Unsupported {
			opts = append(opts, ocagent.UseCompressor(compressionKey))
		} else {
			return nil, nil, &ocTraceExporterError{
				code: errUnsupportedCompressionType,
				msg:  fmt.Sprintf("OpenCensus exporter unsupported compression type %q", ocac.Compression),
			}
		}
	}
	if ocac.CertPemFile != "" {
		creds, err := credentials.NewClientTLSFromFile(ocac.CertPemFile, "")
		if err != nil {
			return nil, nil, &ocTraceExporterError{
				code: errUnableToGetTLSCreds,
				msg:  fmt.Sprintf("OpenCensus exporter unable to read TLS credentials from pem file %q: %v", ocac.CertPemFile, err),
			}
		}
		opts = append(opts, ocagent.WithTLSCredentials(creds))
	} else if ocac.UseSecure {
		certPool, err := x509.SystemCertPool()
		if err != nil {
			return nil, nil, &ocTraceExporterError{
				code: errUnableToGetTLSCreds,
				msg: fmt.Sprintf(
					"OpenCensus exporter unable to read certificates from system pool: %v", err),
			}
		}
		creds := credentials.NewClientTLSFromCert(certPool, "")
		opts = append(opts, ocagent.WithTLSCredentials(creds))
	} else {
		opts = append(opts, ocagent.WithInsecure())
	}
	if len(ocac.Headers) > 0 {
		opts = append(opts, ocagent.WithHeaders(ocac.Headers))
	}
	if ocac.ReconnectionDelay > 0 {
		opts = append(opts, ocagent.WithReconnectionPeriod(ocac.ReconnectionDelay))
	}
	if ocac.KeepaliveParameters != nil {
		opts = append(opts, ocagent.WithGRPCDialOption(grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                ocac.KeepaliveParameters.Time,
			Timeout:             ocac.KeepaliveParameters.Timeout,
			PermitWithoutStream: ocac.KeepaliveParameters.PermitWithoutStream,
		})))
	}

	numWorkers := defaultNumWorkers
	if ocac.NumWorkers > 0 {
		numWorkers = ocac.NumWorkers
	}

	exportersChan := make(chan *ocagent.Exporter, numWorkers)
	for exporterIndex := 0; exporterIndex < numWorkers; exporterIndex++ {
		exporter, serr := ocagent.NewExporter(opts...)
		if serr != nil {
			return nil, nil, fmt.Errorf("cannot configure OpenCensus Trace exporter: %v", serr)
		}
		exportersChan <- exporter
	}

	oce := &ocagentExporter{exporters: exportersChan}
	oexp, err := exporterhelper.NewTraceExporter(
		"oc_trace",
		oce.PushTraceData,
		exporterhelper.WithSpanName("ocservice.exporter.OpenCensus.ConsumeTraceData"),
		exporterhelper.WithRecordMetrics(true))

	if err != nil {
		return nil, nil, err
	}

	return oexp, oce.stop, nil
}

// CreateMetricsExporter creates a metrics exporter based on this config.
func (f *factory) CreateMetricsExporter(logger *zap.Logger, cfg configmodels.Exporter) (consumer.MetricsConsumer, exporter.StopFunc, error) {
	return nil, nil, configerror.ErrDataTypeIsNotSupported
}
