// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal_test

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
)

func TestTimeConverters(t *testing.T) {
	// Ensure that we nanoseconds but that they are also preserved.
	t1 := time.Date(2018, 10, 31, 19, 43, 35, 789, time.UTC)

	assert.EqualValues(t, int64(1541015015000000789), t1.UnixNano())
	tp := internal.TimeToTimestamp(t1)
	assert.EqualValues(t, &timestamp.Timestamp{Seconds: 1541015015, Nanos: 789}, tp)
	assert.EqualValues(t, int64(1541015015000000789), internal.TimestampToTime(tp).UnixNano())
}

func TestUnixNanosConverters(t *testing.T) {
	t1 := time.Date(2020, 03, 24, 1, 13, 23, 789, time.UTC)
	tun := pdata.TimestampUnixNano(t1.UnixNano())

	assert.EqualValues(t, uint64(1585012403000000789), tun)
	tp := internal.UnixNanoToTimestamp(tun)
	assert.EqualValues(t, &timestamp.Timestamp{Seconds: 1585012403, Nanos: 789}, tp)
	assert.EqualValues(t, tun, internal.TimestampToUnixNano(tp))
}
