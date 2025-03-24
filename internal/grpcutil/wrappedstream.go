// Copyright The OpenTelemetry Authors
// Copyright 2016 Michal Witkowski. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package grpcutil // import "go.opentelemetry.io/collector/config/configgrpc"

import (
	"context"

	"google.golang.org/grpc"
)

// this functionality was originally copied from grpc-ecosystem/go-grpc-middleware project

// wrappedServerStream is a thin wrapper around grpc.ServerStream that allows modifying context.
type wrappedServerStream struct {
	grpc.ServerStream
	// wrappedContext is the wrapper's own Context. You can assign it.
	wrappedCtx context.Context
}

// Context returns the wrapper's wrappedContext, overwriting the nested grpc.ServerStream.Context()
func (w *wrappedServerStream) Context() context.Context {
	return w.wrappedCtx
}

// WrapServerStream returns a ServerStream with the new context.
func WrapServerStream(wrappedCtx context.Context, stream grpc.ServerStream) grpc.ServerStream {
	if existing, ok := stream.(*wrappedServerStream); ok {
		existing.wrappedCtx = wrappedCtx
		return existing
	}
	return &wrappedServerStream{ServerStream: stream, wrappedCtx: wrappedCtx}
}
