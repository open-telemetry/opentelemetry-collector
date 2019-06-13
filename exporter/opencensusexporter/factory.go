// Copyright 2018, OpenCensus Authors
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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	"contrib.go.opencensus.io/exporter/ocagent"

	"github.com/census-instrumentation/opencensus-service/consumer"
	"github.com/census-instrumentation/opencensus-service/exporter/exporterhelper"
	"github.com/census-instrumentation/opencensus-service/internal/compression"
	compressiongrpc "github.com/census-instrumentation/opencensus-service/internal/compression/grpc"
	"github.com/census-instrumentation/opencensus-service/internal/configmodels"
	"github.com/census-instrumentation/opencensus-service/internal/factories"
)

var _ = factories.RegisterExporterFactory(&exporterFactory{})

const (
	// The value of "type" key in configuration.
	typeStr = "opencensus"
)

// exporterFactory is the factory for OpenCensus exporter.
type exporterFactory struct {
}

// Type gets the type of the Exporter config created by this factory.
func (f *exporterFactory) Type() string {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for exporter.
func (f *exporterFactory) CreateDefaultConfig() configmodels.Exporter {
	return &ConfigV2{
		ExporterSettings: configmodels.ExporterSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		Headers: map[string]string{},
	}
}

// CreateTraceExporter creates a trace exporter based on this config.
func (f *exporterFactory) CreateTraceExporter(config configmodels.Exporter) (consumer.TraceConsumer, factories.StopFunc, error) {
	ocac := config.(*ConfigV2)

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
func (f *exporterFactory) CreateMetricsExporter(cfg configmodels.Exporter) (consumer.MetricsConsumer, factories.StopFunc, error) {
	return nil, nil, factories.ErrDataTypeIsNotSupported
}
