// Copyright The OpenTelemetry Authors
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

package sharedcomponent

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenthelper"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
)

var id = config.NewID("test")

func TestNewSharedComponents(t *testing.T) {
	comps := NewSharedComponents()
	assert.Len(t, comps.comps, 0)
}

func TestSharedComponents_GetOrAdd(t *testing.T) {
	nop := componenthelper.New()
	createNop := func() component.Component { return nop }

	comps := NewSharedComponents()
	got := comps.GetOrAdd(id, createNop)
	assert.Len(t, comps.comps, 1)
	assert.Same(t, nop, got.Unwrap())
	assert.Same(t, got, comps.GetOrAdd(id, createNop))

	// Shutdown nop will remove
	assert.NoError(t, got.Shutdown(context.Background()))
	assert.Len(t, comps.comps, 0)
	assert.NotSame(t, got, comps.GetOrAdd(id, createNop))
}

func TestSharedComponent(t *testing.T) {
	wantErr := errors.New("my error")
	calledStart := 0
	calledStop := 0
	comp := componenthelper.New(
		componenthelper.WithStart(func(ctx context.Context, host component.Host) error {
			calledStart++
			return wantErr
		}), componenthelper.WithShutdown(func(ctx context.Context) error {
			calledStop++
			return wantErr
		}))
	createComp := func() component.Component { return comp }

	comps := NewSharedComponents()
	got := comps.GetOrAdd(id, createComp)
	assert.Equal(t, wantErr, got.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, 1, calledStart)
	// Second time is not called anymore.
	assert.NoError(t, got.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, 1, calledStart)
	assert.Equal(t, wantErr, got.Shutdown(context.Background()))
	assert.Equal(t, 1, calledStop)
	// Second time is not called anymore.
	assert.NoError(t, got.Shutdown(context.Background()))
	assert.Equal(t, 1, calledStop)
}
