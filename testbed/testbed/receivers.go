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

package testbed

import (
	"context"
	"fmt"
	"log"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/receiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/jaegerreceiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/opencensusreceiver"
)

// Receiver allows to receive traces or metrics. This is an interface that must
// be implemented by all protocols that want to be used in MockBackend.
// Note the terminology: testbed.Receiver is something that can listen and receive data
// from Collector and the corresponding entity in the Collector that sends this data is
// an exporter.
type Receiver interface {
	Start(tc *mockTraceConsumer, mc *mockMetricConsumer) error
	Stop()

	// Generate a config string to place in exporter part of collector config
	// so that it can send data to this receiver.
	GenConfigYAMLStr() string

	// Return protocol name to use in collector config pipeline.
	ProtocolName() string
}

// ReceiverBase implement basic functions needed by all receivers.
type ReceiverBase struct {
	port int
}

func (mb *ReceiverBase) Context() context.Context {
	return context.Background()
}

func (mb *ReceiverBase) ReportFatalError(err error) {
	log.Printf("Fatal error reported: %v", err)
}

// OCReceiver implements OpenCensus format receiver.
type OCReceiver struct {
	ReceiverBase
	receiver *opencensusreceiver.Receiver
}

// Ensure OCReceiver implements MetricExporter.
var _ Receiver = (*OCReceiver)(nil)

const DefaultOCPort = 56565

// NewOCReceiver creates a new OCReceiver that will listen on the specified port after Start
// is called.
func NewOCReceiver(port int) *OCReceiver {
	return &OCReceiver{ReceiverBase: ReceiverBase{port: port}}
}

func (or *OCReceiver) Start(tc *mockTraceConsumer, mc *mockMetricConsumer) error {
	addr := fmt.Sprintf("localhost:%d", or.port)
	var err error
	or.receiver, err = opencensusreceiver.New(addr, tc, mc)
	if err != nil {
		return err
	}

	err = or.receiver.StartTraceReception(or)
	if err != nil {
		return err
	}
	return or.receiver.StartMetricsReception(or)
}

func (or *OCReceiver) Stop() {
	or.receiver.StopTraceReception()
	or.receiver.StopMetricsReception()
}

func (or *OCReceiver) GenConfigYAMLStr() string {
	// Note that this generates an exporter config for agent.
	return fmt.Sprintf(`
  opencensus:
    endpoint: "localhost:%d"`, or.port)
}

func (or *OCReceiver) ProtocolName() string {
	return "opencensus"
}

// jaegerReceiver implements Jaeger format receiver.
type jaegerReceiver struct {
	ReceiverBase
	receiver receiver.TraceReceiver
}

const DefaultJaegerPort = 14268

func NewJaegerReceiver(port int) *jaegerReceiver {
	return &jaegerReceiver{ReceiverBase: ReceiverBase{port: port}}
}

func (jr *jaegerReceiver) Start(tc *mockTraceConsumer, mc *mockMetricConsumer) error {
	jaegerCfg := jaegerreceiver.Configuration{
		CollectorHTTPPort: jr.port,
	}
	var err error
	jr.receiver, err = jaegerreceiver.New(context.Background(), &jaegerCfg, tc, zap.NewNop())
	if err != nil {
		return err
	}

	return jr.receiver.StartTraceReception(jr)
}

func (jr *jaegerReceiver) Stop() {
	if jr.receiver != nil {
		if err := jr.receiver.StopTraceReception(); err != nil {
			log.Printf("Cannot stop Jaeger receiver: %s", err.Error())
		}
	}
}

func (jr *jaegerReceiver) GenConfigYAMLStr() string {
	// Note that this generates an exporter config for agent.
	return fmt.Sprintf(`
  jaeger_thrift_http:
    url: "http://localhost:%d/api/traces"`, jr.port)
}

func (jr *jaegerReceiver) ProtocolName() string {
	return "jaeger_thrift_http"
}
