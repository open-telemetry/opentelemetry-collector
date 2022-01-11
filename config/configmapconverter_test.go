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

package config // import "go.opentelemetry.io/collector/config"

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRetrieved_NilConvertFunc(t *testing.T) {
	mc := NewMapConverter(nil)
	cfgMap := NewMapFromStringMap(map[string]interface{}{"foo": "bar"})
	assert.NoError(t, mc.Convert(context.Background(), cfgMap))
	assert.Equal(t, map[string]interface{}{"foo": "bar"}, cfgMap.ToStringMap())
}

func TestNewConverter_ReturnError(t *testing.T) {
	expectedErr := errors.New("my error")
	mc := NewMapConverter(func(ctx context.Context, m *Map) error {
		return expectedErr
	})
	assert.ErrorIs(t, mc.Convert(context.Background(), NewMap()), expectedErr)
}
