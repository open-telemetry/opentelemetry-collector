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

package configmapprovider // import "go.opentelemetry.io/collector/config/configmapprovider"

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config"
)

func TestNewRetrieved_NilGetFunc(t *testing.T) {
	_, err := NewRetrieved(nil)
	assert.Error(t, err)
}

func TestNewRetrieved_Default(t *testing.T) {
	expectedCfg := config.NewMapFromStringMap(map[string]interface{}{"test": nil})
	expectedErr := errors.New("test")
	ret, err := NewRetrieved(func(context.Context) (*config.Map, error) { return expectedCfg, expectedErr })
	require.NoError(t, err)
	cfg, err := ret.Get(context.Background())
	assert.Equal(t, expectedCfg, cfg)
	assert.Equal(t, expectedErr, err)
	assert.NoError(t, ret.Close(context.Background()))
	// Check that the private func even if called does not panic.
	assert.NotPanics(t, func() { ret.privateRetrieved() })
}

func TestNewRetrieved_WithClose(t *testing.T) {
	expectedCfg := config.NewMapFromStringMap(map[string]interface{}{"test": nil})
	expectedErr := errors.New("test")
	expectedCloseErr := errors.New("test")
	ret, err := NewRetrieved(
		func(context.Context) (*config.Map, error) { return expectedCfg, expectedErr },
		WithClose(func(ctx context.Context) error { return expectedCloseErr }))
	require.NoError(t, err)
	cfg, err := ret.Get(context.Background())
	assert.Equal(t, expectedCfg, cfg)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, expectedCloseErr, ret.Close(context.Background()))
}
