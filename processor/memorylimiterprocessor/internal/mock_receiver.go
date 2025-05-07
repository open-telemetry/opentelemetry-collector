// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/processor/memorylimiterprocessor/internal"

import (
	"context"
	"strings"
	"sync"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/plog"
)

type MockReceiver struct {
	ProduceCount      int
	NextConsumer      consumer.Logs
	lastConsumeResult error
	mux               sync.Mutex
}

func (m *MockReceiver) Start() {
	go m.produce()
}

// This function demonstrates how the receivers should behave when the ConsumeLogs/Traces/Metrics
// call returns an error.
func (m *MockReceiver) produce() {
	for i := 0; i < m.ProduceCount; i++ {
		// Create a large log to consume some memory.
		ld := plog.NewLogs()
		lr := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
		kiloStr := strings.Repeat("x", 10*1024)
		lr.SetSeverityText(kiloStr)

	retry:
		// Send to the pipeline.
		err := m.NextConsumer.ConsumeLogs(context.Background(), ld)

		// Remember the result to be used in the tests.
		m.mux.Lock()
		m.lastConsumeResult = err
		m.mux.Unlock()

		if err != nil {
			// Sending to the pipeline failed.
			if !consumererror.IsPermanent(err) {
				// Retryable error. Try the same data again.
				goto retry
			}
			// Permanent error. Drop it.
		}
	}
}

func (m *MockReceiver) LastConsumeResult() error {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.lastConsumeResult
}
