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

package consumererror

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestThrottleDelay(t *testing.T) {
	err := NewThrottleRetry(time.Minute)
	throttleErr := Throttle{}
	assert.True(t, errors.As(err, &throttleErr))
	assert.Equal(t, throttleErr.RetryDelay(), time.Minute)

	err = fmt.Errorf("%w", err)
	throttleErr = Throttle{}
	assert.True(t, errors.As(err, &throttleErr))
	assert.Equal(t, throttleErr.RetryDelay(), time.Minute)
}
