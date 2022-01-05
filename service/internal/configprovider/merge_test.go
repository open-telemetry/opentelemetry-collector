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

package configprovider

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerge_GetError(t *testing.T) {
	getErr := errors.New("test error")
	pl := NewMerge(&mockProvider{}, &mockProvider{retrieved: newErrGetRetrieved(getErr)})
	require.NotNil(t, pl)
	_, err := pl.Retrieve(context.Background(), nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, getErr)
}

func TestMerge_CloseError(t *testing.T) {
	closeErr := errors.New("test error")
	pl := NewMerge(&mockProvider{}, &mockProvider{retrieved: newErrCloseRetrieved(closeErr)})
	require.NotNil(t, pl)
	cp, err := pl.Retrieve(context.Background(), nil)
	assert.NoError(t, err)
	err = cp.Close(context.Background())
	assert.Error(t, err)
	assert.ErrorIs(t, err, closeErr)
}

func TestMerge_ShutdownError(t *testing.T) {
	shutdownErr := errors.New("test error")
	pl := NewMerge(&mockProvider{}, &mockProvider{shutdownErr: shutdownErr})
	require.NotNil(t, pl)
	err := pl.Shutdown(context.Background())
	assert.Error(t, err)
	assert.ErrorIs(t, err, shutdownErr)
}
