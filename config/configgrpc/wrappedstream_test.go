// Copyright The OpenTelemetry Authors
// Copyright 2016 Michal Witkowski. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configgrpc // import "go.opentelemetry.io/collector/internal/middleware"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

type ctxKey struct{}

var oneCtxKey = ctxKey{}
var otherCtxKey = ctxKey{}

func TestWrapServerStream(t *testing.T) {
	ctx := context.WithValue(context.TODO(), oneCtxKey, 1)
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
