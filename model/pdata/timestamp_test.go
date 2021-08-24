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
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUnixNanosConverters(t *testing.T) {
	t1 := time.Date(2020, 03, 24, 1, 13, 23, 789, time.UTC)
	tun := Timestamp(t1.UnixNano())

	assert.EqualValues(t, uint64(1585012403000000789), tun)
	assert.EqualValues(t, tun, NewTimestampFromTime(t1))
	assert.EqualValues(t, t1, NewTimestampFromTime(t1).AsTime())
}

func TestZeroTimestamp(t *testing.T) {
	assert.Equal(t, time.Unix(0, 0).UTC(), Timestamp(0).AsTime())
	assert.Zero(t, NewTimestampFromTime(time.Unix(0, 0).UTC()))
}
