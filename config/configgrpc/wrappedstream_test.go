// Copyright The OpenTelemetry Authors
// Copyright 2016 Michal Witkowski. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package configgrpc // import "go.opentelemetry.io/collector/internal/middleware"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

type ctxKey struct{}

var (
	oneCtxKey   = ctxKey{}
	otherCtxKey = ctxKey{}
)

func TestWrapServerStream(t *testing.T) {
	ctx := context.WithValue(t.Context(), oneCtxKey, 1)
	fake := &fakeServerStream{ctx: ctx}
	assert.NotNil(t, fake.Context().Value(oneCtxKey), "values from fake must propagate to wrapper")
	wrapped := wrapServerStream(context.WithValue(fake.Context(), otherCtxKey, 2), fake)
	assert.NotNil(t, wrapped.Context().Value(oneCtxKey), "values from wrapper must be set")
	assert.NotNil(t, wrapped.Context().Value(otherCtxKey), "values from wrapper must be set")
}

func TestDoubleWrapping(t *testing.T) {
	fake := &fakeServerStream{ctx: context.Background()}
	wrapped := wrapServerStream(fake.Context(), fake)
	assert.Same(t, wrapped, wrapServerStream(wrapped.Context(), wrapped)) // should be noop
	assert.Equal(t, fake, wrapped.ServerStream)
}

type fakeServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f *fakeServerStream) Context() context.Context {
	return f.ctx
}
