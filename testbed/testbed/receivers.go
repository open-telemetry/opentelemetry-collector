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

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/receiver/jaegerreceiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/opencensusreceiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/otlpreceiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/zipkinreceiver"
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

func (mb *DataReceiverBase) ReportFatalError(err error) {
	log.Printf("Fatal error reported: %v", err)
}

// GetFactory of the specified kind. Returns the factory for a component type.
func (mb *DataReceiverBase) GetFactory(kind component.Kind, componentType string) component.Factory {
	return nil
}

// Return map of extensions. Only enabled and created extensions will be returned.
func (mb *DataReceiverBase) GetExtensions() map[configmodels.Extension]component.ServiceExtension {
	return nil
}

// OCDataReceiver implements OpenCensus format receiver.
type OCDataReceiver struct {
	DataReceiverBase
	receiver *opencensusreceiver.Receiver
}

// Ensure OCDataReceiver implements DataReceiver.
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
	or.receiver, err = opencensusreceiver.New("opencensus", "tcp", addr, tc, mc)
	if err != nil {
		return err
	}

	return or.receiver.Start(context.Background(), or)
}

func (or *OCDataReceiver) Stop() {
	or.receiver.Shutdown(context.Background())
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
	receiver component.TraceReceiver
}

const DefaultJaegerPort = 14250

func NewJaegerDataReceiver(port int) *JaegerDataReceiver {
	return &JaegerDataReceiver{DataReceiverBase: DataReceiverBase{Port: port}}
}

func (jr *JaegerDataReceiver) Start(tc *MockTraceConsumer, mc *MockMetricConsumer) error {
	jaegerCfg := jaegerreceiver.Configuration{
		CollectorGRPCPort: jr.Port,
	}
	var err error
	params := component.ReceiverCreateParams{Logger: zap.NewNop()}
	jr.receiver, err = jaegerreceiver.New("jaeger", &jaegerCfg, tc, params)
	if err != nil {
		return err
	}

	return jr.receiver.Start(context.Background(), jr)
}

func (jr *JaegerDataReceiver) Stop() {
	if jr.receiver != nil {
		if err := jr.receiver.Shutdown(context.Background()); err != nil {
			log.Printf("Cannot stop Jaeger receiver: %s", err.Error())
		}
	}
}

func (jr *JaegerDataReceiver) GenConfigYAMLStr() string {
	// Note that this generates an exporter config for agent.
	return fmt.Sprintf(`
  jaeger:
    endpoint: "localhost:%d"`, jr.Port)
}

func (jr *JaegerDataReceiver) ProtocolName() string {
	return "jaeger"
}

// OTLPDataReceiver implements OTLP format receiver.
type OTLPDataReceiver struct {
	DataReceiverBase
	receiver *otlpreceiver.Receiver
}

// Ensure OTLPDataReceiver implements DataReceiver.
var _ DataReceiver = (*OTLPDataReceiver)(nil)

const DefaultOTLPPort = 55680

// NewOTLPDataReceiver creates a new OTLPDataReceiver that will listen on the specified port after Start
// is called.
func NewOTLPDataReceiver(port int) *OTLPDataReceiver {
	return &OTLPDataReceiver{DataReceiverBase: DataReceiverBase{Port: port}}
}

func (or *OTLPDataReceiver) Start(tc *MockTraceConsumer, mc *MockMetricConsumer) error {
	addr := fmt.Sprintf("localhost:%d", or.Port)
	var err error
	or.receiver, err = otlpreceiver.New("otlp", "tcp", addr, tc, mc)
	if err != nil {
		return err
	}

	return or.receiver.Start(context.Background(), or)
}

func (or *OTLPDataReceiver) Stop() {
	or.receiver.Shutdown(context.Background())
}

func (or *OTLPDataReceiver) GenConfigYAMLStr() string {
	// Note that this generates an exporter config for agent.
	return fmt.Sprintf(`
  otlp:
    endpoint: "localhost:%d"`, or.Port)
}

func (or *OTLPDataReceiver) ProtocolName() string {
	return "otlp"
}

// ZipkinDataReceiver implements Zipkin format receiver.
type ZipkinDataReceiver struct {
	DataReceiverBase
	receiver *zipkinreceiver.ZipkinReceiver
}

const DefaultZipkinAddressPort = 9411

func NewZipkinDataReceiver(port int) *ZipkinDataReceiver {
	return &ZipkinDataReceiver{DataReceiverBase: DataReceiverBase{Port: port}}
}

func (zr *ZipkinDataReceiver) Start(tc *MockTraceConsumer, mc *MockMetricConsumer) error {
	var err error
	address := fmt.Sprintf("localhost:%d", zr.Port)
	zr.receiver, err = zipkinreceiver.New("zipkin", address, tc)

	if err != nil {
		return err
	}

	return zr.receiver.Start(context.Background(), zr)
}

func (zr *ZipkinDataReceiver) Stop() {
	if zr.receiver != nil {
		if err := zr.receiver.Shutdown(context.Background()); err != nil {
			log.Printf("Cannot stop Zipkin receiver: %s", err.Error())
		}
	}
}

func (zr *ZipkinDataReceiver) GenConfigYAMLStr() string {
	// Note that this generates an exporter config for agent.
	return fmt.Sprintf(`
  zipkin:
    url: http://localhost:%d/api/v2/spans
    format: json`, zr.Port)
}

func (zr *ZipkinDataReceiver) ProtocolName() string {
	return "zipkin"
}
