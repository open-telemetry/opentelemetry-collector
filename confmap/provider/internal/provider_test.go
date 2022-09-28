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

package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
)

func TestNewRetrievedFromYAML(t *testing.T) {
	ret, err := NewRetrievedFromYAML([]byte{})
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, confmap.New(), retMap)
	assert.NoError(t, ret.Close(context.Background()))
}

func TestNewRetrievedFromYAMLWithOptions(t *testing.T) {
	want := errors.New("my error")
	ret, err := NewRetrievedFromYAML([]byte{}, confmap.WithRetrievedClose(func(context.Context) error { return want }))
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, confmap.New(), retMap)
	assert.Equal(t, want, ret.Close(context.Background()))
}

func TestNewRetrievedFromYAMLInvalidYAMLBytes(t *testing.T) {
	_, err := NewRetrievedFromYAML([]byte("[invalid:,"))
	assert.Error(t, err)
}

func TestNewRetrievedFromYAMLInvalidAsMap(t *testing.T) {
	ret, err := NewRetrievedFromYAML([]byte("string"))
	require.NoError(t, err)

	_, err = ret.AsConf()
	assert.Error(t, err)
}
