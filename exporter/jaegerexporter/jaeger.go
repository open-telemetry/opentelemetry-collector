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
	"context"
	"fmt"
	"sync"

	"github.com/spf13/viper"

	"contrib.go.opencensus.io/exporter/jaeger"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-service/exporter/exporterwrapper"
	"github.com/open-telemetry/opentelemetry-service/internal"
)

type jaegerExporter struct {
	counter   uint32
	exporters chan *jaeger.Exporter
}

type jTraceExporterErrorCode int
type jTraceExporterError struct {
	code jTraceExporterErrorCode
	msg  string
}

var _ error = (*jTraceExporterError)(nil)

func (e *jTraceExporterError) Error() string {
	return e.msg
}

const (
	defaultNumWorkers int = 2

	_ jTraceExporterErrorCode = iota // skip 0
	// errCollectorEndpointRequired indicates that this exporter was not provided with a collector endpoint in its config.
	errCollectorEndpointRequired

	// errUsernameRequired indicates that this exporter was not provided with a user name in its config.
	errUsernameRequired

	// errPasswordRequired indicates that this exporter was not provided with a password in its config.
	errPasswordRequired

	// errAlreadyStopped indicates that the exporter was already stopped.
	errAlreadyStopped
)

// Slight modified version of go/src/contrib.go.opencensus.io/exporter/jaeger/jaeger.go
type jaegerConfig struct {
	CollectorEndpoint string `mapstructure:"collector_endpoint,omitempty"`
	Username          string `mapstructure:"username,omitempty"`
	Password          string `mapstructure:"password,omitempty"`
	ServiceName       string `mapstructure:"service_name,omitempty"`
}

// JaegerExportersFromViper unmarshals the viper and returns exporter.TraceExporters targeting
// Jaeger according to the configuration settings.
func JaegerExportersFromViper(v *viper.Viper) (tps []consumer.TraceConsumer, mps []consumer.MetricsConsumer, doneFns []func() error, err error) {
	var cfg struct {
		Jaeger *jaegerConfig `mapstructure:"jaeger"`
	}
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, nil, nil, err
	}
	jc := cfg.Jaeger
	if jc == nil {
		return nil, nil, nil, nil
	}

	// jaeger.NewExporter performs configurqtion validation
	je, err := jaeger.NewExporter(jaeger.Options{
		CollectorEndpoint: jc.CollectorEndpoint,
		Username:          jc.Username,
		Password:          jc.Password,
		Process: jaeger.Process{
			ServiceName: jc.ServiceName,
		},
	})
	if err != nil {
		return nil, nil, nil, err
	}

	doneFns = append(doneFns, func() error {
		je.Flush()
		return nil
	})

	jte, err := exporterwrapper.NewExporterWrapper("jaeger", "ocservice.exporter.Jaeger.ConsumeTraceData", je)
	if err != nil {
		return nil, nil, nil, err
	}
	// TODO: Examine "contrib.go.opencensus.io/exporter/jaeger" to see
	// if trace.ExportSpan was constraining and if perhaps the Jaeger
	// upload can use the context and information from the Node.
	tps = append(tps, jte)
	return
}

func (j *jaegerExporter) stop() error {
	wg := &sync.WaitGroup{}
	var errors []error
	var errorsMu sync.Mutex
	visitedCnt := 0
	for currExporter := range j.exporters {
		wg.Add(1)
		go func(exporter *j.Exporter) {
			defer wg.Done()
			err := exporter.Stop()
			if err != nil {
				errorsMu.Lock()
				errors = append(errors, err)
				errorsMu.Unlock()
			}
		}(currExporter)
		visitedCnt++
		if visitedCnt == cap(j.exporters) {
			// Visited and started Stop on all exporters, just wait for the stop to finish.
			break
		}
	}

	wg.Wait()
	close(j.exporters)

	return internal.CombineErrors(errors)
}

func (j *jaegerExporter) PushTraceData(ctx context.Context, td consumerdata.TraceData) (int, error) {
	// Get first available exporter.
	exporter, ok := <-j.exporters
	if !ok {
		err := &jTraceExporterError{
			code: errAlreadyStopped,
			msg:  fmt.Sprintf("Jaeger exporter was already stopped."),
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
	j.exporters <- exporter
	if err != nil {
		return len(td.Spans), err
	}
	return 0, nil
}
