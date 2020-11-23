// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fluentforwardreceiver

import (
	"context"

	"go.opencensus.io/stats"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/fluentforwardreceiver/observ"
)

// Collector acts as an aggregator of LogRecords so that we don't have to
// generate as many pdata.Logs instances...we can pre-batch the LogRecord
// instances from several Forward events into one to hopefully reduce
// allocations and GC overhead.
type Collector struct {
	nextConsumer consumer.LogsConsumer
	eventCh      <-chan Event
	logger       *zap.Logger
}

func newCollector(eventCh <-chan Event, next consumer.LogsConsumer, logger *zap.Logger) *Collector {
	return &Collector{
		nextConsumer: next,
		eventCh:      eventCh,
		logger:       logger,
	}
}

func (c *Collector) Start(ctx context.Context) {
	go c.processEvents(ctx)
}

func (c *Collector) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-c.eventCh:
			buffered := []Event{e}
			// Pull out anything waiting on the eventCh to get better
			// efficiency on LogResource allocations.
			buffered = fillBufferUntilChanEmpty(c.eventCh, buffered)

			logs := collectLogRecords(buffered)
			c.nextConsumer.ConsumeLogs(ctx, logs)
		}
	}
}

func fillBufferUntilChanEmpty(eventCh <-chan Event, buf []Event) []Event {
	for {
		select {
		case e2 := <-eventCh:
			buf = append(buf, e2)
		default:
			return buf
		}
	}
}

func collectLogRecords(events []Event) pdata.Logs {
	out := pdata.NewLogs()

	logs := out.ResourceLogs()

	logs.Resize(1)
	rls := logs.At(0)

	rls.InstrumentationLibraryLogs().Resize(1)
	logSlice := rls.InstrumentationLibraryLogs().At(0).Logs()

	for i := range events {
		events[i].LogRecords().MoveAndAppendTo(logSlice)
	}

	stats.Record(context.Background(), observ.RecordsGenerated.M(int64(out.LogRecordCount())))

	return out
}
