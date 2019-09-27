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

	"github.com/open-telemetry/opentelemetry-collector/internal"
)

func TestTimeConverters(t *testing.T) {
	// Ensure that we nanoseconds but that they are also preserved.
	t1 := time.Date(2018, 10, 31, 19, 43, 35, 789, time.UTC)
	ts := internal.TimeToTimestamp(t1)
	t2 := time.Unix(ts.Seconds, int64(ts.Nanos))

	// Verification for paranoia
	if g, w := int64(1541015015000000789), t1.UnixNano(); g != w {
		t.Errorf("InitialTime time nanos mismatch\nGot: %d\nWant:%d", g, w)
	}
	if g, w := int64(1541015015000000789), t2.UnixNano(); g != w {
		t.Errorf("Convertedback time nanos mismatch\nGot: %d\nWant:%d", g, w)
	}
	if g, w := t1.UnixNano(), t2.UnixNano(); g != w {
		t.Errorf("Convertedback time does not match original time\nGot: %d\nWant:%d", g, w)
	}
}
