// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package memory // import "go.opentelemetry.io/collector/processor/memorylimiterprocessor/internal/memory"

import (
	"runtime"
)

type Stats = runtime.MemStats

// StatsFunc defines a means of loading the current memory stats
// that can be overridden within tests.
type StatsFunc func(*Stats)

func (fn StatsFunc) Load(ballastSize uint64) (*Stats, bool) {
	ms := new(Stats)
	fn.Update(ms)
	mismatched := ms.Alloc < ballastSize
	if ms.Alloc >= ballastSize {
		ms.Alloc -= ballastSize
	}
	return ms, mismatched
}

func (fn StatsFunc) Update(ms *Stats) {
	if fn != nil {
		fn(ms)
	} else {
		runtime.ReadMemStats(ms)
	}
}
