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
)

func TestNewOptionalDouble(t *testing.T) {
	optionalDouble := NewOptionalDouble(10.9)
	assert.EqualValues(t, 10.9, optionalDouble.orig.Value)
}

func TestEqual(t *testing.T) {
	optionalDouble := NewOptionalDouble(10.8)
	other := NewOptionalDouble(10.8)
	assert.True(t, optionalDouble.Equal(other))

	other = NewOptionalDouble(11)
	assert.False(t, optionalDouble.Equal(other))
}

func TestSize(t *testing.T) {
	opt := NewOptionalDouble(10.8)
	assert.Equal(t, 9, opt.Size())
	assert.Equal(t, 0, (*OptionalDouble)(nil).Size())
}

func TestMarshal(t *testing.T) {
	optionalDouble := NewOptionalDouble(10.8)
	marshaled := make([]byte, 9)
	len, _ := optionalDouble.MarshalTo(marshaled)
	assert.Equal(t, 9, len)
	assert.Equal(t, []byte{0x09, 0x9a, 0x99, 0x99, 0x99, 0x99, 0x99, 0x25, 0x40}, marshaled)
}

func TestUnmarshal(t *testing.T) {
	optionalDouble := OptionalDouble{}
	err := optionalDouble.Unmarshal([]byte{0x09, 0x9a, 0x99, 0x99, 0x99, 0x99, 0x99, 0x25, 0x40})
	assert.Nil(t, err)
	assert.Equal(t, 10.8, optionalDouble.orig.Value)
	assert.False(t, optionalDouble.IsEmpty())

	optionalDouble = OptionalDouble{}
	err = optionalDouble.Unmarshal([]byte{})
	assert.Nil(t, err)
	assert.Equal(t, float64(0), optionalDouble.orig.Value)
	assert.True(t, optionalDouble.IsEmpty())
}
