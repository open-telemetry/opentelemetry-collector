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
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	"contrib.go.opencensus.io/exporter/ocagent"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	"github.com/census-instrumentation/opencensus-service/consumer"
	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/exporter/exporterhelper"
	"github.com/census-instrumentation/opencensus-service/internal"
	"github.com/census-instrumentation/opencensus-service/internal/compression"
	compressiongrpc "github.com/census-instrumentation/opencensus-service/internal/compression/grpc"
)

// keepaliveConfig exposes the keepalive.ClientParameters to be used by the exporter.
// Refer to the original data-structure for the meaning of each parameter.
type keepaliveConfig struct {
	Time                time.Duration `mapstructure:"time,omitempty"`
	Timeout             time.Duration `mapstructure:"timeout,omitempty"`
	PermitWithoutStream bool          `mapstructure:"permit-without-stream,omitempty"`
}

type opencensusConfig struct {
	Endpoint            string            `mapstructure:"endpoint,omitempty"`
	Compression         string            `mapstructure:"compression,omitempty"`
	Headers             map[string]string `mapstructure:"headers,omitempty"`
	NumWorkers          int               `mapstructure:"num-workers,omitempty"`
	CertPemFile         string            `mapstructure:"cert-pem-file,omitempty"`
	UseSecure           bool              `mapstructure:"secure,omitempty"`
	ReconnectionDelay   time.Duration     `mapstructure:"reconnection-delay,omitempty"`
	KeepaliveParameters *keepaliveConfig  `mapstructure:"keepalive,omitempty"`
	// TODO: service name options.
}

type ocagentExporter struct {
	counter   uint32
	exporters chan *ocagent.Exporter
}

type ocTraceExporterErrorCode int
type ocTraceExporterError struct {
	code ocTraceExporterErrorCode
	msg  string
}

var _ error = (*ocTraceExporterError)(nil)

func (e *ocTraceExporterError) Error() string {
	return e.msg
}

const (
	defaultNumWorkers int = 2

	_ ocTraceExporterErrorCode = iota // skip 0
	// errEndpointRequired indicates that this exporter was not provided with an endpoint in its config.
	errEndpointRequired
	// errUnsupportedCompressionType indicates that this exporter was provided with a compression protocol it does not support.
	errUnsupportedCompressionType
	// errUnableToGetTLSCreds indicates that this exporter could not read the provided TLS credentials.
	errUnableToGetTLSCreds
	// errAlreadyStopped indicates that the exporter was already stopped.
	errAlreadyStopped
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
		return nil, nil, nil, &ocTraceExporterError{
			code: errEndpointRequired,
			msg:  "OpenCensus exporter config requires an Endpoint",
		}
	}

	opts := []ocagent.ExporterOption{ocagent.WithAddress(ocac.Endpoint)}
	if ocac.Compression != "" {
		if compressionKey := compressiongrpc.GetGRPCCompressionKey(ocac.Compression); compressionKey != compression.Unsupported {
			opts = append(opts, ocagent.UseCompressor(compressionKey))
		} else {
			return nil, nil, nil, &ocTraceExporterError{
				code: errUnsupportedCompressionType,
				msg:  fmt.Sprintf("OpenCensus exporter unsupported compression type %q", ocac.Compression),
			}
		}
	}
	if ocac.CertPemFile != "" {
		creds, err := credentials.NewClientTLSFromFile(ocac.CertPemFile, "")
		if err != nil {
			return nil, nil, nil, &ocTraceExporterError{
				code: errUnableToGetTLSCreds,
				msg:  fmt.Sprintf("OpenCensus exporter unable to read TLS credentials from pem file %q: %v", ocac.CertPemFile, err),
			}
		}
		opts = append(opts, ocagent.WithTLSCredentials(creds))
	} else if ocac.UseSecure {
		certPool, err := x509.SystemCertPool()
		if err != nil {
			return nil, nil, nil, &ocTraceExporterError{
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
			return nil, nil, nil, fmt.Errorf("cannot configure OpenCensus Trace exporter: %v", serr)
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
		return nil, nil, nil, err
	}

	tps = append(tps, oexp)
	doneFns = append(doneFns, oce.stop)

	// TODO: (@odeke-em, @songya23) implement ExportMetrics for OpenCensus.
	// mps = append(mps, oexp)
	return
}

func (oce *ocagentExporter) stop() error {
	wg := &sync.WaitGroup{}
	var errors []error
	var errorsMu sync.Mutex
	visitedCnt := 0
	for currExporter := range oce.exporters {
		wg.Add(1)
		go func(exporter *ocagent.Exporter) {
			defer wg.Done()
			err := exporter.Stop()
			if err != nil {
				errorsMu.Lock()
				errors = append(errors, err)
				errorsMu.Unlock()
			}
		}(currExporter)
		visitedCnt++
		if visitedCnt == cap(oce.exporters) {
			// Visited and started Stop on all exporters, just wait for the stop to finish.
			break
		}
	}

	wg.Wait()
	close(oce.exporters)

	return internal.CombineErrors(errors)
}

func (oce *ocagentExporter) PushTraceData(ctx context.Context, td data.TraceData) (int, error) {
	// Get first available exporter.
	exporter, ok := <-oce.exporters
	if !ok {
		err := &ocTraceExporterError{
			code: errAlreadyStopped,
			msg:  fmt.Sprintf("OpenCensus exporter was already stopped."),
		}
		return len(td.Spans), err
	}

	err := exporter.ExportTraceServiceRequest(
		&agenttracepb.ExportTraceServiceRequest{
			Spans:    td.Spans,
			Resource: td.Resource,
			Node:     td.Node,
		},
	)
	oce.exporters <- exporter
	if err != nil {
		return len(td.Spans), err
	}
	return 0, nil
}
