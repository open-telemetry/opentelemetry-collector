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
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestUnixNanosConverters(t *testing.T) {
	ts := time.Date(2020, 03, 24, 1, 13, 23, 789, time.UTC)
	event := NewSpanEvent()
	event.SetTime(ts)
	assert.EqualValues(t, uint64(1585012403000000789), event.orig.TimeUnixNano)
	assert.EqualValues(t, ts, event.Time())
	assert.EqualValues(t, &timestamppb.Timestamp{Seconds: 1585012403, Nanos: 789}, timestamppb.New(event.Time()))
}

func TestZeroTimestamps(t *testing.T) {
	ts := time.Date(1970, 01, 1, 0, 0, 0, 0, time.UTC)
	event := NewSpanEvent()
	event.SetTime(ts)
	assert.EqualValues(t, uint64(0), event.orig.TimeUnixNano)
	assert.EqualValues(t, ts, event.Time())
	assert.EqualValues(t, &timestamppb.Timestamp{Seconds: 0, Nanos: 0}, timestamppb.New(event.Time()))
}
