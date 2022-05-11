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

package status

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/config"
)

func TestNewEventMinimal(t *testing.T) {
	ev := NewEvent(OK)
	assert.Equal(t, OK, ev.Type())
	assert.NotEqual(t, 0, ev.Timestamp())
	assert.Equal(t, config.ComponentID{}, ev.ComponentID())
	assert.Nil(t, ev.Err())
}

func TestNewEventAllOptions(t *testing.T) {
	expectedComponentID := config.NewComponentID("nop")
	expectedTimestamp := time.Now().UnixNano()
	expectedErr := errors.New("expect this!")
	ev := NewEvent(
		RecoverableError,
		WithComponentID(expectedComponentID),
		WithTimestamp(expectedTimestamp),
		WithError(expectedErr),
	)
	assert.Equal(t, RecoverableError, ev.Type())
	assert.Equal(t, expectedComponentID, ev.ComponentID())
	assert.Equal(t, expectedTimestamp, ev.Timestamp())
	assert.Equal(t, expectedErr, ev.Err())
}
