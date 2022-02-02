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

package data // import "go.opentelemetry.io/collector/model/internal/data"

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionalDoublePresent(t *testing.T) {
	field := NewOptionalDouble(10.4)
	assert.False(t, field.IsEmpty())
	assert.Equal(t, 9, field.Size())
	data := make([]byte, field.Size())
	len, err := field.MarshalTo(data)
	require.NoError(t, err)
	require.Equal(t, 9, len)
	require.Equal(t, []byte{0x0, 0xcd, 0xcc, 0xcc, 0xcc, 0xcc, 0xcc, 0x24, 0x40}, data)
	field = OptionalDouble{}
	err = field.Unmarshal([]byte{0xcd, 0xcc, 0xcc, 0xcc, 0xcc, 0xcc, 0x24, 0x40})
	require.NoError(t, err)
	require.Equal(t, 10.4, field.DoubleValue)
}

func TestOptionalDoubleNotPresent(t *testing.T) {
	field := OptionalDouble{}
	assert.True(t, field.IsEmpty())
	assert.Equal(t, 0, field.Size())
	data := make([]byte, field.Size())
	len, err := field.MarshalTo(data)
	require.NoError(t, err)
	require.Equal(t, 0, len)
	require.Equal(t, []byte{}, data)
}
