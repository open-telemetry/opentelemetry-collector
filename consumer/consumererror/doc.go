// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package consumererror provides wrappers to easily classify errors. This allows
// appropriate action by error handlers without the need to know each individual
// error type/instance. These errors are returned by the Consume*() functions of
// the consumer interfaces.
//
// # Error handling
//
// The consumererror package provides a way to classify errors into two categories: Permanent and
// NonPermanent. The Permanent errors are those that are not expected to be resolved by retrying the
// same data.
//
// If the error is Permanent, then the Consume*() call should not be retried with the same data.
// This typically happens when the data cannot be serialized by the exporter that is attached to the
// pipeline or when the destination refuses the data because it cannot decode it.
//
// If the error is non-Permanent then the Consume*() call should be retried with the same data. This
// may be done by the component itself, however typically it is done by the original sender, after
// the receiver in the pipeline returns a response to the sender indicating that the Collector is
// currently overloaded and the request must be retried.
package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"
