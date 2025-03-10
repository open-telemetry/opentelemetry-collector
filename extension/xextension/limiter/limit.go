// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiter // import "go.opentelemetry.io/collector/extension/xextension/limiter"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

// Extension is the interface that storage extensions must implement
type Extension interface {
	extension.Extension

	// GetLimiter will create a client for use by the specified
	// component.  The component can use the client to limit
	// admission to the pipeline.
	GetLimiter(ctx context.Context, kind component.Kind, id component.ID) (Limiter, error)
}

// Limiter implements a limiter for byte-weighted request admission.
type Limiter interface {
	// Acquire asks the controller to admit the caller.
	//
	// The weight parameter specifies how large of an request to
	// consider in the limiter.  This should be a measure of the
	// size of the request after it is uncompressed.  One way to
	// compute the weight of a request is byte its OTLP encoding
	// size using the appropriate pdata `Sizer` type.  However,
	// components are encouraged to use an approximation, not
	// necessarily based on the OTLP encoding size, if there is an
	// efficient substitute.
	//
	// The goal is for the estimated weight to approximate the
	// amount of real memory that will be occupied while the
	// request is in flight.  If the component uses a protocol
	// with less repetition than OTLP, it is possible for the
	// Sizer to overestimate the amount of real memory used,
	// because Golang's immutable string value allows
	// it. Therefore, components should prefer an inexpensive,
	// approximate method to determine weight in bytes.
	//
	// The limiter is permitted to block the request. After making
	// its decision, the return value is exclusively a function
	// value or an error. Admit will return when one of the
	// following events occurs:
	//
	//   (1) admission is allowed, or
	//   (2) the provided ctx becomes canceled, or
	//   (3) the limiter determines that the request should fail.
	//
	// In case (1), the return value will be a non-nil
	// ReleaseFunc. The caller must invoke it after it is finished
	// with the resource being guarded by the admission
	// controller.
	//
	// In case (2), the return value should be similar to:
	// - gRPC: Cancelled or DeadlineExceeded error
	// - HTTP: Status 408.
	//
	// In case (3), the return value should be similar to:
	// - gRPC: ResourceExhausted error
	// - HTTP: Status 429.
	Acquire(ctx context.Context, weight uint64) (ReleaseFunc, error)
}

// ReleaseFunc is returned by Acquire when the Acquire() was admitted.
type ReleaseFunc func()
