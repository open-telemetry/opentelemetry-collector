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
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	"github.com/spf13/viper"
	"google.golang.org/grpc/credentials"

	"github.com/census-instrumentation/opencensus-service/consumer"
	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/exporter/exporterhelper"
	"github.com/census-instrumentation/opencensus-service/internal/compression"
	"github.com/census-instrumentation/opencensus-service/internal/compression/grpc"
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

// OpenCensusTraceExportersFromViper unmarshals the viper and returns an consumer.TraceConsumer targeting
// OpenCensus Agent/Collector according to the configuration settings.
func OpenCensusTraceExportersFromViper(v *viper.Viper) (tps []consumer.TraceConsumer, mps []consumer.MetricsConsumer, doneFns []func() error, err error) {
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

	oce := &ocagentExporter{exporters: exporters}
	oexp, err := exporterhelper.NewTraceExporter(
		"oc_trace",
		oce.PushTraceData,
		exporterhelper.WithSpanName("ocservice.exporter.OpenCensus.ConsumeTraceData"),
		exporterhelper.WithRecordMetrics(true))

	if err != nil {
		return nil, nil, nil, err
	}

	tps = append(tps, oexp)

	// TODO: (@odeke-em, @songya23) implement ExportMetrics for OpenCensus.
	// mps = append(mps, oexp)
	return
}

func (oce *ocagentExporter) PushTraceData(ctx context.Context, td data.TraceData) (int, error) {
	// Get an exporter worker round-robin
	exporter := oce.exporters[atomic.AddUint32(&oce.counter, 1)%uint32(len(oce.exporters))]
	err := exporter.ExportTraceServiceRequest(
		&agenttracepb.ExportTraceServiceRequest{
			Spans:    td.Spans,
			Resource: td.Resource,
			Node:     td.Node,
		},
	)
	if err != nil {
		return len(td.Spans), err
	}
	return 0, nil
}
