// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"

// DroppedItemsErr is a non-failure sentinel that an exporter's push function
// can return to signal that it intentionally dropped a number of items rather
// than failing to send them. The error is not propagated to the rest of the
// pipeline. exporterhelper subtracts the dropped count from
// exporter_sent_<signal> and records it on exporter_dropped_<signal> instead.
//
// Typical use is an exporter that receives data it cannot translate to the
// destination format (e.g. the Prometheus exporter dropping non-monotonic
// DELTA sums) and wants the operator's internal telemetry to reflect a drop
// rather than a successful send.
type DroppedItemsErr = experr.DroppedItemsErr

// NewDroppedItemsErr creates a [DroppedItemsErr] for the given count and an
// optional human-readable reason. The reason is recorded as the
// exporter.dropped.reason metric attribute at detailed telemetry level only,
// to keep cardinality bounded at lower levels.
func NewDroppedItemsErr(dropped int, reason string) error {
	return experr.NewDroppedItemsErr(dropped, reason)
}
