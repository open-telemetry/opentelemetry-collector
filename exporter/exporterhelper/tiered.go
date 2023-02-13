// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

type tieredSender struct {
	name    string
	id      component.ID
	signal  component.DataType
	logger  *zap.Logger
	primary *queuedRetrySender
	backlog *queuedRetrySender
}

func newTieredSender(
	id component.ID,
	signal component.DataType,
	qCfg QueueSettings,
	bCfg QueueSettings,
	rCfg RetrySettings,
	reqUnmarshaler internal.RequestUnmarshaler,
	nextSender requestSender,
	logger *zap.Logger,
) *tieredSender {
	name := id.String()
	traceAttr := attribute.String(obsmetrics.ExporterKey, name)

	ts := &tieredSender{
		name:   name,
		id:     id,
		signal: signal,
		logger: logger,
	}

	ts.primary = newQueuedRetrySender(name, id, signal, traceAttr, qCfg, rCfg, reqUnmarshaler, nextSender, logger)
	if qCfg.Enabled && bCfg.Enabled {
		name = fmt.Sprintf("%s:backlog", name)
		ts.backlog = newQueuedRetrySender(name, id, signal, traceAttr, bCfg, rCfg, reqUnmarshaler, nextSender, logger)
		ts.primary.requeueSender = ts.backlog
	}
	return ts
}

// start is invoked during service startup.
func (ts *tieredSender) start(ctx context.Context, host component.Host) error {
	if err := ts.primary.start(ctx, host); err != nil {
		return err
	}
	if ts.backlog != nil {
		if err := ts.backlog.start(ctx, host); err != nil {
			return err
		}
		ts.backlog.queue.OnOverflow(func(req internal.Request) {
			ts.logger.Error(
				"Backlog is full. Dropping data.",
				zap.Int("dropped_items", req.Count()),
			)
		})
		ts.primary.queue.OnOverflow(func(req internal.Request) {
			if err := ts.backlog.send(req); err != nil {
				ts.logger.Error(
					"Unable to add overflow to backlog. Dropping data.",
					zap.Error(err),
					zap.Int("dropped_items", req.Count()),
				)
			}
		})
	}
	return nil
}

// shutdown is invoked during service shutdown.
func (ts *tieredSender) shutdown() {
	ts.primary.shutdown()
	if ts.backlog != nil {
		ts.backlog.requeuingEnabled = false
		ts.backlog.shutdown()
	}
}

// send the request to the primary sender.
func (ts *tieredSender) send(req internal.Request) error {
	return ts.primary.send(req)
}

// wrapConsumerSender calls the wrap function on the primary's consumer sender
// and sets it as the consumer sender for both primary and backlog if it is enabled.
func (ts *tieredSender) wrapConsumerSender(wrap func(consumer requestSender) requestSender) {
	ts.primary.consumerSender = wrap(ts.primary.consumerSender)
	if ts.backlog != nil {
		ts.backlog.consumerSender = wrap(ts.backlog.consumerSender)
	}
}
