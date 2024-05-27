// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package conslogtest // import "go.opentelemetry.io/collector/consumer/conslog/conslogtest"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/consumer/conslog"
	"go.opentelemetry.io/collector/consumer/internal/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
)

// LogsSink is a conslog.Logs that acts like a sink that
// stores all logs and allows querying them for testing.
type LogsSink struct {
	consumertest.NonMutatingConsumer
	mu             sync.Mutex
	logs           []plog.Logs
	logRecordCount int
}

var _ conslog.Logs = (*LogsSink)(nil)

// ConsumeLogs stores logs to this sink.
func (sle *LogsSink) ConsumeLogs(_ context.Context, ld plog.Logs) error {
	sle.mu.Lock()
	defer sle.mu.Unlock()

	sle.logs = append(sle.logs, ld)
	sle.logRecordCount += ld.LogRecordCount()

	return nil
}

// AllLogs returns the logs stored by this sink since last Reset.
func (sle *LogsSink) AllLogs() []plog.Logs {
	sle.mu.Lock()
	defer sle.mu.Unlock()

	copyLogs := make([]plog.Logs, len(sle.logs))
	copy(copyLogs, sle.logs)
	return copyLogs
}

// LogRecordCount returns the number of log records stored by this sink since last Reset.
func (sle *LogsSink) LogRecordCount() int {
	sle.mu.Lock()
	defer sle.mu.Unlock()
	return sle.logRecordCount
}

// Reset deletes any stored data.
func (sle *LogsSink) Reset() {
	sle.mu.Lock()
	defer sle.mu.Unlock()

	sle.logs = nil
	sle.logRecordCount = 0
}
