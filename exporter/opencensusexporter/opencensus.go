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
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"contrib.go.opencensus.io/exporter/ocagent"
	"github.com/spf13/viper"
	"google.golang.org/grpc/credentials"

	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/internal/compression"
	"github.com/census-instrumentation/opencensus-service/internal/compression/grpc"
	"github.com/census-instrumentation/opencensus-service/observability"
	"github.com/census-instrumentation/opencensus-service/processor"
)

type opencensusConfig struct {
	Endpoint    string            `mapstructure:"endpoint,omitempty"`
	Compression string            `mapstructure:"compression,omitempty"`
	Headers     map[string]string `mapstructure:"headers,omitempty"`
	NumWorkers  int               `mapstructure:"num-workers,omitempty"`
	CertPemFile string            `mapstructure:"cert-pem-file,omitempty"`

	// TODO: add insecure, service name options.
}

type ocagentExporter struct {
	counter   uint32
	exporters []*ocagent.Exporter
}

const (
	defaultNumWorkers int = 2
)

var (
	// ErrEndpointRequired indicates that this exporter was not provided with an endpoint in its config.
	ErrEndpointRequired = errors.New("OpenCensus exporter config requires an Endpoint")
	// ErrUnsupportedCompressionType indicates that this exporter was provided with a compression protocol it does not support.
	ErrUnsupportedCompressionType = errors.New("OpenCensus exporter unsupported compression type")
	// ErrUnableToGetTLSCreds indicates that this exporter could not read the provided TLS credentials.
	ErrUnableToGetTLSCreds = errors.New("OpenCensus exporter unable to read TLS credentials")
)

var _ processor.TraceDataProcessor = (*ocagentExporter)(nil)

// OpenCensusTraceExportersFromViper unmarshals the viper and returns an processor.TraceDataProcessor targeting
// OpenCensus Agent/Collector according to the configuration settings.
func OpenCensusTraceExportersFromViper(v *viper.Viper) (tdps []processor.TraceDataProcessor, mdps []processor.MetricsDataProcessor, doneFns []func() error, err error) {
	var cfg struct {
		OpenCensus *opencensusConfig `mapstructure:"opencensus"`
	}
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, nil, nil, err
	}
	ocac := cfg.OpenCensus
	if ocac == nil {
		return nil, nil, nil, nil
	}

	if ocac.Endpoint == "" {
		return nil, nil, nil, ErrEndpointRequired
	}

	opts := []ocagent.ExporterOption{ocagent.WithAddress(ocac.Endpoint)}
	if ocac.Compression != "" {
		if compressionKey := grpc.GetGRPCCompressionKey(ocac.Compression); compressionKey != compression.Unsupported {
			opts = append(opts, ocagent.UseCompressor(compressionKey))
		} else {
			return nil, nil, nil, ErrUnsupportedCompressionType
		}
	}
	if ocac.CertPemFile != "" {
		creds, err := credentials.NewClientTLSFromFile(ocac.CertPemFile, "")
		if err != nil {
			return nil, nil, nil, ErrUnableToGetTLSCreds
		}
		opts = append(opts, ocagent.WithTLSCredentials(creds))
	} else {
		opts = append(opts, ocagent.WithInsecure())
	}
	if len(ocac.Headers) > 0 {
		opts = append(opts, ocagent.WithHeaders(ocac.Headers))
	}

	numWorkers := defaultNumWorkers
	if ocac.NumWorkers > 0 {
		numWorkers = ocac.NumWorkers
	}

	exporters := make([]*ocagent.Exporter, 0, numWorkers)
	for exporterIndex := 0; exporterIndex < numWorkers; exporterIndex++ {
		exporter, serr := ocagent.NewExporter(opts...)
		if serr != nil {
			return nil, nil, nil, fmt.Errorf("cannot configure OpenCensus Trace exporter: %v", serr)
		}
		exporters = append(exporters, exporter)
		doneFns = append(doneFns, func() error {
			exporter.Flush()
			return nil
		})
	}

	oexp := &ocagentExporter{exporters: exporters}
	tdps = append(tdps, oexp)

	// TODO: (@odeke-em, @songya23) implement ExportMetrics for OpenCensus.
	// mdps = append(mdps, oexp)
	return
}

const exporterTagValue = "oc_trace"

func (oce *ocagentExporter) ProcessTraceData(ctx context.Context, td data.TraceData) error {
	// Get an exporter worker round-robin
	exporter := oce.exporters[atomic.AddUint32(&oce.counter, 1)%uint32(len(oce.exporters))]
	err := exporter.ExportTraceServiceRequest(
		&agenttracepb.ExportTraceServiceRequest{
			Spans:    td.Spans,
			Resource: td.Resource,
			Node:     td.Node,
		},
	)
	ctxWithExporterName := observability.ContextWithExporterName(ctx, exporterTagValue)
	if err != nil {
		// TODO: If failed to send all maybe record a different metric. Failed to "Sent", but
		// this may not be accurate if we have retry outside of this exporter. Maybe the retry
		// processor should record these metrics. For the moment we assume no retry.
		observability.RecordTraceExporterMetrics(ctxWithExporterName, len(td.Spans), len(td.Spans))
		return err
	}
	observability.RecordTraceExporterMetrics(ctxWithExporterName, len(td.Spans), 0)
	return nil
}
