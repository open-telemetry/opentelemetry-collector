// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package extensionlimiter implements a resource-based request limit
// interface for pipeline components.
package extensionlimiter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

// Extension is a component for resource-based request limits.
type Extension interface {
	extension.Extension

	Limiter
}

// Settings describe additional details of the request.
type Settings struct {
	// Kind describes which signal is being used.  This field permits
	// a Limiter to discriminate or prioritize by signal.
	Kind component.Kind

	// Potential additions:
	// - component.ID of the calling component
	// - protocol/method of the request
}

// Weight is the argument to a limiter. Any of three fields may be
// set; if a field is zero, no limit will apply. Receivers are meant
// to call a limiter one or more times as this information becomes
// available.
//
// For example, consider an unary gRPC in a protocol that encapsulates
// another protocol body, such that middleware only sees the number of
// compressed bytes per request.  When the gRPC interceptor runs it
// will:
//
//	rel, err := Acquire(ctx, Settings{...}, Weight{
//	  Requests: 1,
//	  Records: 0,
//	  Bytes: len(compressedData),
//	})
//
// Later, when the data is uncompressed and the actual record count
// is known, the receiver should call the limiter again:
//
//	rel, err := Acquire(ctx, Settings{...}, Weight{
//	  Requests: 0,
//	  Records: recordNum,
//	  Bytes: len(uncompressedData),
//	})
//
// or, if the code has dropped all references to the compressed data
// it could call instead:
//
//	rel, err := Acquire(ctx, Settings{...}, Weight{
//	  Requests: 0,
//	  Records: recordNum,
//	  Bytes: len(uncompressedData) - len(compressedData),
//	})
//
// Limiters should ignore zero values in the Weight struct and they
// should ignore values they are not concerned with limiting.
// Therefore, in the examples above, if a limiter is based on request
// count or record count, they are guaranteed one call to the limiter
// will be a no-op because either Requests or Records is zero in one
// of the two calls. On the other hand, for this example, if the
// limiter is based on bytes there will be two real calls to the
// limiter, to account for the two non-zero Bytes fields.
type Weight struct {
	// Requests corresponds with external request number.  Typically
	// this field value is acquired with value 1 once per request.
	Requests int

	// Records corresponds with the number of logs, spans, metric
	// points, and profiles in a request.
	Records int

	// Bytes corresponds with the size of the request in bytes.
	// This number can be approximate, however it should be an
	// estimate of the amount of real memory that will be used
	// while the data resides in the pipeline, so should use the
	// uncompressed size of the data.  As a default method, the
	// corresponding pdata Sizer method may be used.
	Bytes int
}

// Limiter is the interface for imposing resource-based request limits.
type Limiter interface {
	// Acquire asks the controller to admit the caller based on
	// context, Settings, and Weight parameters.
	//
	// The Weight.Bytes field is especially important for
	// memory-based limiters.  This field may be approximated.
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
	//   (1) admission is allowed (error == nil), or
	//   (2) the provided ctx becomes canceled (error != nil), or
	//   (3) the limiter determines that the request should fail (error != nil).
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
	Acquire(ctx context.Context, settings Settings, weight Weight) (ReleaseFunc, error)
}

// ReleaseFunc is returned on success from Acquire. Callers will
// typically defer a call to the release function until the resources
// that were acquired are no longer in use.
type ReleaseFunc func()

// TODO: Add WithStart(), WithShutdown() options etc.
