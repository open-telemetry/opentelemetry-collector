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

// Package fanoutconsumer contains implementations of Traces/Metrics/Logs consumers
// that fan out the data to multiple other consumers.
package fanoutconsumer // import "go.opentelemetry.io/collector/service/internal/fanoutconsumer"

import (
	"context"
	"fmt"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
)

// NewLogs wraps multiple log consumers in a single one.
// It fanouts the incoming data to all the consumers, and does smart routing:
//   - Clones only to the consumer that needs to mutate the data.
//   - If all consumers needs to mutate the data one will get the original data.
func NewLogs(lcs []consumer.Logs) consumer.Logs {
	if len(lcs) == 1 {
		// Don't wrap if no need to do it.
		return lcs[0]
	}
	var pass []consumer.Logs
	var clone []consumer.Logs
	for i := 0; i < len(lcs)-1; i++ {
		if !lcs[i].Capabilities().MutatesData {
			pass = append(pass, lcs[i])
		} else {
			clone = append(clone, lcs[i])
		}
	}
	// Give the original data to the last consumer if no other read-only consumer,
	// otherwise put it in the right bucket. Never share the same data between
	// a mutating and a non-mutating consumer since the non-mutating consumer may process
	// data async and the mutating consumer may change the data before that.
	if len(pass) == 0 || !lcs[len(lcs)-1].Capabilities().MutatesData {
		pass = append(pass, lcs[len(lcs)-1])
	} else {
		clone = append(clone, lcs[len(lcs)-1])
	}
	return &logsConsumer{pass: pass, clone: clone}
}

type logsConsumer struct {
	pass  []consumer.Logs
	clone []consumer.Logs
}

func (lsc *logsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeLogs exports the plog.Logs to all consumers wrapped by the current one.
func (lsc *logsConsumer) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	var errs error
	// Initially pass to clone exporter to avoid the case where the optimization of sending
	// the incoming data to a mutating consumer is used that may change the incoming data before
	// cloning.
	for _, lc := range lsc.clone {
		clonedLogs := plog.NewLogs()
		ld.CopyTo(clonedLogs)
		errs = multierr.Append(errs, lc.ConsumeLogs(ctx, clonedLogs))
	}
	for _, lc := range lsc.pass {
		errs = multierr.Append(errs, lc.ConsumeLogs(ctx, ld))
	}
	return errs
}

var _ connector.LogsRouter = (*logsRouter)(nil)

type logsRouter struct {
	consumer.Logs
	consumers map[component.ID]consumer.Logs
}

func NewLogsRouter(cm map[component.ID]consumer.Logs) consumer.Logs {
	consumers := make([]consumer.Logs, 0, len(cm))
	for _, consumer := range cm {
		consumers = append(consumers, consumer)
	}
	return &logsRouter{
		Logs:      NewLogs(consumers),
		consumers: cm,
	}
}

func (r *logsRouter) PipelineIDs() []component.ID {
	ids := make([]component.ID, 0, len(r.consumers))
	for id := range r.consumers {
		ids = append(ids, id)
	}
	return ids
}

func (r *logsRouter) Consumer(pipelineIDs ...component.ID) (consumer.Logs, error) {
	if len(pipelineIDs) == 0 {
		return nil, fmt.Errorf("missing consumers")
	}
	consumers := make([]consumer.Logs, 0, len(pipelineIDs))
	var errors error
	for _, pipelineID := range pipelineIDs {
		c, ok := r.consumers[pipelineID]
		if ok {
			consumers = append(consumers, c)
		} else {
			errors = multierr.Append(errors, fmt.Errorf("missing consumer: %q", pipelineID))
		}
	}
	if errors != nil {
		// TODO potentially this could return a NewLogs with the valid consumers
		return nil, errors
	}
	return NewLogs(consumers), nil
}
