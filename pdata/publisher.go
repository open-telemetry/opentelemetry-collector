// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/pdata"

import (
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type ComponentID string

// Publisher is used by components to publish data to the RemoteTap extension.
type Publisher interface {
	// IsActive returns true when at least one connection is open for the given componentID.
	IsActive(ComponentID) bool
	// PublishMetrics sends metrics for a given componentID.
	PublishMetrics(ComponentID, pmetric.Metrics)
	// PublishTraces sends traces for a given componentID.
	PublishTraces(ComponentID, ptrace.Traces)
	// PublishLogs sends logs for a given componentID.
	PublishLogs(ComponentID, plog.Logs)
}
