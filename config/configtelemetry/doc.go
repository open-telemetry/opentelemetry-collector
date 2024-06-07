// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package configtelemetry defines various telemetry level for configuration.
// It enables every component to have access to telemetry level
// to enable metrics only when necessary.
//
// 1. configtelemetry.None
//
// No telemetry data should be collected.
//
// 2. configtelemetry.Basic
//
// Signals associated with this level cover the essential coverage of the component telemetry.
//
// This is the default level recommended when running the collector.
//
// The signals associated with this level must have low cardinality.
//
// The signals associated with this level must represent a small data volume:
//   - No more than 5 metrics reported.
//   - At most 1 span actively recording simultaneously, covering the critical path.
//   - At most 5 log records produced every 30s.
//
// Not all signals defined in the component telemetry are active.
//
// 3. configtelemetry.Normal
//
// Signals associated with this level cover the complete coverage of the component telemetry.
//
// The signals associated with this level must control cardinality.
//
// The signals associated with this level must represent a controlled data volume:
//   - No more than 20 metrics reported.
//   - At most 5 spans actively recording simultaneously.
//   - At most 20 log records produced every 30s.
//
// All signals defined in the component telemetry are active.
//
// 4. configtelemetry.Detailed
//
// Signals associated with this level cover the complete coverage of the component telemetry.
//
// The signals associated with this level may exhibit high cardinality.
//
// There is no limit on data volume.
//
// All signals defined in the component telemetry are active.
package configtelemetry // import "go.opentelemetry.io/collector/config/configtelemetry"
