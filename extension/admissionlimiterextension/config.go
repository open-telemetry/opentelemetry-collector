// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package admissionlimiterextension // import "go.opentelemetry.io/collector/extension/admissionlimiterextension"

// Config is the basis of a memory limiter that counts the
// number of bytes pending and in the pipeline.
type Config struct {
	// RequestLimitMiB limits the number of requests that are received by the stream based on
	// uncompressed request size. Request size is used to control how much traffic we admit
	// for processing.  When this field is zero, admission control is disabled meaning all
	// requests will be immediately accepted.
	RequestLimitMiB uint64 `mapstructure:"request_limit_mib"`

	// WaitingLimitMiB is the limit on the amount of data waiting to be consumed.
	// This is a dimension of memory limiting to ensure waiters are not consuming an
	// unexpectedly large amount of memory in receivers that use it.
	WaitingLimitMiB uint64 `mapstructure:"waiting_limit_mib"`
}
