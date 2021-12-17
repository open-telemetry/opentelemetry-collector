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

package configmapprovider

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config"
)

func TestMerge_GetError(t *testing.T) {
	pl := NewMerge(&errProvider{err: nil}, &errProvider{err: errors.New("my error")})
	require.NotNil(t, pl)
	cp, err := pl.Retrieve(context.Background(), nil)
	assert.Error(t, err)
	assert.Nil(t, cp)
}

func TestMerge_CloseError(t *testing.T) {
	pl := NewMerge(&errProvider{err: nil}, &errProvider{closeErr: errors.New("my error")})
	require.NotNil(t, pl)
	cp, err := pl.Retrieve(context.Background(), nil)
	assert.NoError(t, err)
	assert.Error(t, cp.Close(context.Background()))
}

func TestMerge_ShutdownError(t *testing.T) {
	pl := NewMerge(&errProvider{err: nil}, &errProvider{err: errors.New("my error")})
	require.NotNil(t, pl)
	assert.Error(t, pl.Shutdown(context.Background()))
}

type errProvider struct {
	err      error
	closeErr error
}

func (epl *errProvider) Retrieve(context.Context, func(*ChangeEvent)) (Retrieved, error) {
	if epl.err == nil {
		return &simpleRetrieved{confMap: config.NewMap(), closeFunc: func(context.Context) error { return epl.closeErr }}, nil
	}
	return nil, epl.err
}

func (epl *errProvider) Shutdown(context.Context) error {
	return epl.err
}
