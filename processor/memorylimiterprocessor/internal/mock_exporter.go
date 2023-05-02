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

package internal // import "go.opentelemetry.io/collector/processor/memorylimiterprocessor/internal"

import (
	"context"
	"sync/atomic"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
)

type MockExporter struct {
	destAvailable     int64
	acceptedLogCount  int64
	deliveredLogCount int64
	Logs              []plog.Logs
}

var _ consumer.Logs = (*MockExporter)(nil)

func (e *MockExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}

func (e *MockExporter) ConsumeLogs(_ context.Context, ld plog.Logs) error {
	atomic.AddInt64(&e.acceptedLogCount, int64(ld.LogRecordCount()))

	if atomic.LoadInt64(&e.destAvailable) == 1 {
		// Destination is available, immediately deliver.
		atomic.AddInt64(&e.deliveredLogCount, int64(ld.LogRecordCount()))
	} else {
		// Destination is not available. Queue the logs in the exporter.
		e.Logs = append(e.Logs, ld)
	}
	return nil
}

func (e *MockExporter) SetDestAvailable(available bool) {
	if available {
		// Pretend we delivered all queued accepted logs.
		atomic.AddInt64(&e.deliveredLogCount, atomic.LoadInt64(&e.acceptedLogCount))

		// Get rid of the delivered logs so that memory can be collected.
		e.Logs = nil

		// Now mark destination available so that subsequent ConsumeLogs
		// don't queue the logs anymore.
		atomic.StoreInt64(&e.destAvailable, 1)

	} else {
		atomic.StoreInt64(&e.destAvailable, 0)
	}
}

func (e *MockExporter) AcceptedLogCount() int {
	return int(atomic.LoadInt64(&e.acceptedLogCount))
}

func (e *MockExporter) DeliveredLogCount() int {
	return int(atomic.LoadInt64(&e.deliveredLogCount))
}

func NewMockExporter() *MockExporter {
	return &MockExporter{}
}
