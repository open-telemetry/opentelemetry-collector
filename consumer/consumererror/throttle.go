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

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

import (
	"time"
)

type Throttle struct {
	delay time.Duration
}

func (t Throttle) Error() string {
	return "Throttle (" + t.delay.String() + ")"
}

func (t Throttle) RetryDelay() time.Duration {
	return t.delay
}

// NewThrottleRetry creates a new throttle retry error.
func NewThrottleRetry(delay time.Duration) error {
	return Throttle{
		delay: delay,
	}
}
