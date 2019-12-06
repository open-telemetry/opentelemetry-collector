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
	"os"
	"sync"
	"sync/atomic"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/receiver"
)

// MockBackend is a backend that allows receiving the data locally.
type MockBackend struct {
	// Metric and trace consumers
	tc *mockTraceConsumer
	mc *mockMetricConsumer

	receiver Receiver

	// Log file
	logFilePath string
	logFile     *os.File

	// Start/stop flags
	isStarted bool
	stopOnce  sync.Once
}

// NewMockBackend creates a new mock backend that receives data using specified receiver.
func NewMockBackend(logFilePath string, receiver Receiver) *MockBackend {
	mb := &MockBackend{
		logFilePath: logFilePath,
		receiver:    receiver,
		tc:          &mockTraceConsumer{},
		mc:          &mockMetricConsumer{},
	}
	return mb
}

var _ receiver.Host = (*MockBackend)(nil)

func (mb *MockBackend) Context() context.Context {
	return context.Background()
}

func (mb *MockBackend) ReportFatalError(err error) {
	log.Printf("Fatal error reported: %v", err)
}

// Start a backend.
func (mb *MockBackend) Start() error {
	log.Printf("Starting mock backend...")

	var err error

	// Open log file
	mb.logFile, err = os.Create(mb.logFilePath)
	if err != nil {
		return err
	}

	err = mb.receiver.Start(mb.tc, mb.mc)
	if err != nil {
		return err
	}

	mb.isStarted = true
	return nil
}

// Stop the backend
func (mb *MockBackend) Stop() {
	mb.stopOnce.Do(func() {
		if !mb.isStarted {
			return
		}

		log.Printf("Stopping mock backend...")

		mb.logFile.Close()
		mb.receiver.Stop()

		// Print stats.
		log.Printf("Stopped backend. %s", mb.GetStats())
	})
}

func (mb *MockBackend) GetStats() string {
	return fmt.Sprintf("Received:%5d spans", mb.SpansReceived())
}

func (mb *MockBackend) SpansReceived() uint64 {
	return atomic.LoadUint64(&mb.tc.spansReceived)
}

type mockTraceConsumer struct {
	spansReceived uint64
}

func (tc *mockTraceConsumer) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	atomic.AddUint64(&tc.spansReceived, uint64(len(td.Spans)))

	for _, span := range td.Spans {
		var spanSeqnum int64
		var traceSeqnum int64

		seqnumAttr, ok := span.Attributes.AttributeMap["load_generator.span_seq_num"]
		if ok {
			spanSeqnum = seqnumAttr.GetIntValue()
		}

		seqnumAttr, ok = span.Attributes.AttributeMap["load_generator.trace_seq_num"]
		if ok {
			traceSeqnum = seqnumAttr.GetIntValue()
		}

		// Ignore the seqnums for now. We will use them later.
		_ = spanSeqnum
		_ = traceSeqnum
	}

	return nil
}

type mockMetricConsumer struct {
}

func (mc *mockMetricConsumer) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	// Metrics backend is not implemented yet.
	return nil
}
