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

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
)

// NewLogs wraps multiple log consumers in a single one sending the data marked as shared.
func NewLogs(lcs []consumer.Logs) consumer.Logs {
	if len(lcs) == 1 {
		// Don't wrap if no need to do it.
		return lcs[0]
	}
	return &logsConsumer{consumers: lcs}
}

type logsConsumer struct {
	consumers []consumer.Logs
}

// ConsumeLogs exports the plog.Logs to all consumers wrapped by the current one.
func (lsc *logsConsumer) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	var errs error
	for _, lc := range lsc.consumers {
		// Send logs marked as shared so that they are cloned if mutation is needed.
		errs = multierr.Append(errs, lc.ConsumeLogs(ctx, ld.AsShared()))
	}
	return errs
}
