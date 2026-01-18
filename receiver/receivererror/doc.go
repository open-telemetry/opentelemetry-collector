// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package receivererror provides error types for receivers.
//
// The PartialReceiveError type allows receivers to report partial success
// when processing data. This enables accurate observability metrics that
// distinguish between:
//   - Full success (no error)
//   - Partial success (error with specific failure count)
//   - Full failure (regular error, all items failed)
package receivererror // import "go.opentelemetry.io/collector/receiver/receivererror"
