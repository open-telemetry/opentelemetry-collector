// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

// Package queuebatchprocessor implements a processor that queues and batches
// telemetry using the exporterhelper queue/batch implementation. It is the
// replacement for the batchprocessor described in
// docs/rfcs/batching-migration.md.
package queuebatchprocessor // import "go.opentelemetry.io/collector/processor/queuebatchprocessor"
