// Copyright The OpenTelemetry Authors
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
	"fmt"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configmodels"
)

const (
	// The value of "type" key in configuration.
	typeStr = "opencensus"
)

// Factory is the factory for OpenCensus exporter.
type Factory struct {
}

// Type gets the type of the Exporter config created by this factory.
func (f *Factory) Type() configmodels.Type {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for exporter.
func (f *Factory) CreateDefaultConfig() configmodels.Exporter {
	return &Config{
		ExporterSettings: configmodels.ExporterSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Headers: map[string]string{},
		},
	}
}

// CreateTraceExporter creates a trace exporter based on this config.
func (f *Factory) CreateTraceExporter(logger *zap.Logger, config configmodels.Exporter) (component.TraceExporterOld, error) {
	ocac := config.(*Config)
	opts, err := f.OCAgentOptions(logger, ocac)
	if err != nil {
		return nil, err
	}
	return NewTraceExporter(logger, config, opts...)
}

// OCAgentOptions takes the oc exporter Config and generates ocagent Options
func (f *Factory) OCAgentOptions(logger *zap.Logger, ocac *Config) ([]ocagent.ExporterOption, error) {
	if ocac.Endpoint == "" {
		return nil, &ocExporterError{
			code: errEndpointRequired,
			msg:  "OpenCensus exporter config requires an Endpoint",
		}
	}
	// TODO(ccaraman): Clean up this usage of gRPC settings apart of PR to address issue #933.
	opts := []ocagent.ExporterOption{ocagent.WithAddress(ocac.Endpoint)}
	if ocac.Compression != "" {
		if compressionKey := configgrpc.GetGRPCCompressionKey(ocac.Compression); compressionKey != configgrpc.CompressionUnsupported {
			opts = append(opts, ocagent.UseCompressor(compressionKey))
		} else {
			return nil, &ocExporterError{
				code: errUnsupportedCompressionType,
				msg:  fmt.Sprintf("OpenCensus exporter unsupported compression type %q", ocac.Compression),
			}
		}
	}
	if ocac.TLSSetting.CAFile != "" {
		creds, err := credentials.NewClientTLSFromFile(ocac.TLSSetting.CAFile, "")
		if err != nil {
			return nil, &ocExporterError{
				code: errUnableToGetTLSCreds,
				msg:  fmt.Sprintf("OpenCensus exporter unable to read TLS credentials from pem file %q: %v", ocac.TLSSetting.CAFile, err),
			}
		}
		opts = append(opts, ocagent.WithTLSCredentials(creds))
	} else if !ocac.TLSSetting.Insecure {
		tlsConf, err := ocac.TLSSetting.LoadTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("OpenCensus exporter failed to load TLS config: %w", err)
		}
		creds := credentials.NewTLS(tlsConf)
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
	if ocac.Keepalive != nil {
		opts = append(opts, ocagent.WithGRPCDialOption(grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                ocac.Keepalive.Time,
			Timeout:             ocac.Keepalive.Timeout,
			PermitWithoutStream: ocac.Keepalive.PermitWithoutStream,
		})))
	}
	return opts, nil
}

// CreateMetricsExporter creates a metrics exporter based on this config.
func (f *Factory) CreateMetricsExporter(logger *zap.Logger, config configmodels.Exporter) (component.MetricsExporterOld, error) {
	oCfg := config.(*Config)
	opts, err := f.OCAgentOptions(logger, oCfg)
	if err != nil {
		return nil, err
	}
	return NewMetricsExporter(logger, config, opts...)
}
