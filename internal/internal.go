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

package internal

import (
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

// TimeToTimestamp converts a time.Time to a timestamp.Timestamp pointer.
func TimeToTimestamp(t time.Time) *timestamp.Timestamp {
	if t.IsZero() {
		return nil
	}
	nanoTime := t.UnixNano()
	return &timestamp.Timestamp{
		Seconds: nanoTime / 1e9,
		Nanos:   int32(nanoTime % 1e9),
	}
}

func TimestampToTime(ts *timestamp.Timestamp) (t time.Time) {
	if ts == nil {
		return
	}
	return time.Unix(ts.Seconds, int64(ts.Nanos))
}

func TimestampToUnixNano(ts *timestamp.Timestamp) (t pdata.TimestampUnixNano) {
	if ts == nil {
		return
	}
	return pdata.TimestampUnixNano(uint64(TimestampToTime(ts).UnixNano()))
}

func UnixNanoToTimestamp(u pdata.TimestampUnixNano) *timestamp.Timestamp {
	return TimeToTimestamp(time.Unix(0, int64(u)))
}
