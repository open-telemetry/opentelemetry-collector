// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package confmap

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRetrievedFromMap(t *testing.T) {
	conf := New()
	ret := NewRetrievedFromMap(conf)
	retMap, err := ret.AsMap()
	require.NoError(t, err)
	assert.Same(t, conf, retMap)
	assert.NoError(t, ret.Close(context.Background()))
}

func TestNewRetrievedFromMapWithOptions(t *testing.T) {
	want := errors.New("my error")
	conf := New()
	ret := NewRetrievedFromMap(conf, WithRetrievedClose(func(context.Context) error { return want }))
	retMap, err := ret.AsMap()
	require.NoError(t, err)
	assert.Same(t, conf, retMap)
	assert.Equal(t, want, ret.Close(context.Background()))
}
