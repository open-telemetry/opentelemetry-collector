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

// Timestamp is a time specified as UNIX Epoch time in nanoseconds since
// 00:00:00 UTC on 1 January 1970.
type Timestamp uint64

// TimestampFromTime returns a new Timestamp converted from time.Time.
func TimestampFromTime(t time.Time) Timestamp {
	// 0 is a special case and want to make sure we return zero timestamp.
	if t.IsZero() {
		return 0
	}
	return Timestamp(uint64(t.UnixNano()))
}

// ToTime converts this Timestamp to time.Time.
func (tun Timestamp) ToTime() time.Time {
	// 0 is a special case and want to make sure we return a time that IsZero() returns true.
	if tun == 0 {
		return time.Time{}
	}
	return time.Unix(0, int64(tun)).UTC()
}

func (tun Timestamp) String() string {
	return time.Unix(0, int64(tun)).String()
}

// TODO Remove everything below.

func TimestampToUnixNano(ts *timestamppb.Timestamp) (t Timestamp) {
	if ts == nil {
		return
	}
	return Timestamp(uint64(ts.AsTime().UnixNano()))
}

func UnixNanoToTimestamp(u Timestamp) *timestamppb.Timestamp {
	// 0 is a special case and want to make sure we return nil.
	if u == 0 {
		return nil
	}
	return timestamppb.New(u.ToTime())
}
