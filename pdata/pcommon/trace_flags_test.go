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

package pcommon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTraceFlags(t *testing.T) {
	flags := TraceFlags(1)
	assert.True(t, flags.IsSampled())
	assert.EqualValues(t, uint8(1), flags)

	flags = flags.WithIsSampled(false)
	assert.False(t, flags.IsSampled())
	assert.EqualValues(t, uint8(0), flags)

	flags = flags.WithIsSampled(true)
	assert.True(t, flags.IsSampled())
	assert.EqualValues(t, uint8(1), flags)
}

func TestDefaultTraceFlags(t *testing.T) {
	flags := DefaultTraceFlags
	assert.False(t, flags.IsSampled())
	assert.EqualValues(t, uint8(0), flags)
}
