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

package pdata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpanID(t *testing.T) {
	sid := InvalidSpanID()
	assert.EqualValues(t, [8]byte{}, sid.Bytes())
	assert.True(t, sid.IsEmpty())

	sid = NewSpanID([8]byte{1, 2, 3, 4, 4, 3, 2, 1})
	assert.EqualValues(t, [8]byte{1, 2, 3, 4, 4, 3, 2, 1}, sid.Bytes())
	assert.False(t, sid.IsEmpty())
}
