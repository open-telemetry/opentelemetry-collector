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
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TimestampToUnixNano(ts *timestamppb.Timestamp) (t TimestampUnixNano) {
	if ts == nil {
		return
	}
	return TimestampUnixNano(uint64(ts.AsTime().UnixNano()))
}

func UnixNanoToTimestamp(u TimestampUnixNano) *timestamppb.Timestamp {
	// 0 is a special case and want to make sure we return nil.
	if u == 0 {
		return nil
	}
	return timestamppb.New(UnixNanoToTime(u))
}

func UnixNanoToTime(u TimestampUnixNano) time.Time {
	// 0 is a special case and want to make sure we return a time that IsZero() returns true.
	if u == 0 {
		return time.Time{}
	}
	return time.Unix(0, int64(u)).UTC()
}

func TimeToUnixNano(t time.Time) TimestampUnixNano {
	// 0 is a special case and want to make sure we return zero timestamp to support inverse function for UnixNanoToTime
	if t.IsZero() {
		return 0
	}
	return TimestampUnixNano(uint64(t.UnixNano()))
}
