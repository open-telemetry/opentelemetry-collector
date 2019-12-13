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

// DataReceiver allows to receive traces or metrics. This is an interface that must
// be implemented by all protocols that want to be used in MockBackend.
// Note the terminology: testbed.DataReceiver is something that can listen and receive data
// from Collector and the corresponding entity in the Collector that sends this data is
// an exporter.
type DataReceiver interface {
	Start(tc *MockTraceConsumer, mc *MockMetricConsumer) error
	Stop()

	// Generate a config string to place in exporter part of collector config
	// so that it can send data to this receiver.
	GenConfigYAMLStr() string

	// Return protocol name to use in collector config pipeline.
	ProtocolName() string
}

// DataReceiverBase implement basic functions needed by all receivers.
type DataReceiverBase struct {
	// Port on which to listen.
	Port int
}

func (mb *DataReceiverBase) Context() context.Context {
	return context.Background()
}

func (mb *DataReceiverBase) ReportFatalError(err error) {
	log.Printf("Fatal error reported: %v", err)
}

// OCDataReceiver implements OpenCensus format receiver.
type OCDataReceiver struct {
	DataReceiverBase
	receiver *opencensusreceiver.Receiver
}

// Ensure OCDataReceiver implements MetricDataSender.
var _ DataReceiver = (*OCDataReceiver)(nil)

const DefaultOCPort = 56565

// NewOCDataReceiver creates a new OCDataReceiver that will listen on the specified port after Start
// is called.
func NewOCDataReceiver(port int) *OCDataReceiver {
	return &OCDataReceiver{DataReceiverBase: DataReceiverBase{Port: port}}
}

func (or *OCDataReceiver) Start(tc *MockTraceConsumer, mc *MockMetricConsumer) error {
	addr := fmt.Sprintf("localhost:%d", or.Port)
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

func (or *OCDataReceiver) Stop() {
	or.receiver.StopTraceReception()
	or.receiver.StopMetricsReception()
}

func (or *OCDataReceiver) GenConfigYAMLStr() string {
	// Note that this generates an exporter config for agent.
	return fmt.Sprintf(`
  opencensus:
    endpoint: "localhost:%d"`, or.Port)
}

func (or *OCDataReceiver) ProtocolName() string {
	return "opencensus"
}

// JaegerDataReceiver implements Jaeger format receiver.
type JaegerDataReceiver struct {
	DataReceiverBase
	receiver receiver.TraceReceiver
}

const DefaultJaegerPort = 14268

func NewJaegerDataReceiver(port int) *JaegerDataReceiver {
	return &JaegerDataReceiver{DataReceiverBase: DataReceiverBase{Port: port}}
}

func (jr *JaegerDataReceiver) Start(tc *MockTraceConsumer, mc *MockMetricConsumer) error {
	jaegerCfg := jaegerreceiver.Configuration{
		CollectorHTTPPort: jr.Port,
	}
	var err error
	jr.receiver, err = jaegerreceiver.New(context.Background(), &jaegerCfg, tc, zap.NewNop())
	if err != nil {
		return err
	}

	return jr.receiver.StartTraceReception(jr)
}

func (jr *JaegerDataReceiver) Stop() {
	if jr.receiver != nil {
		if err := jr.receiver.StopTraceReception(); err != nil {
			log.Printf("Cannot stop Jaeger receiver: %s", err.Error())
		}
	}
}

func (jr *JaegerDataReceiver) GenConfigYAMLStr() string {
	// Note that this generates an exporter config for agent.
	return fmt.Sprintf(`
  jaeger_thrift_http:
    url: "http://localhost:%d/api/traces"`, jr.Port)
}

func (jr *JaegerDataReceiver) ProtocolName() string {
	return "jaeger_thrift_http"
}
