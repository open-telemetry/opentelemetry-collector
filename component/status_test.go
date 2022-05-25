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

package component

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/config"
)

func TestNewEventMinimal(t *testing.T) {
	ev, err := NewStatusEvent(OK)
	assert.NoError(t, err)
	assert.Equal(t, OK, ev.Type())
	assert.NotEqual(t, 0, ev.Timestamp())
	assert.Equal(t, config.ComponentID{}, ev.ComponentID())
	assert.Nil(t, ev.Err())
}

func TestNewEventAllOptions(t *testing.T) {
	expectedComponentID := config.NewComponentID("nop")
	expectedTimestamp := time.Now()
	expectedErr := errors.New("expect this")
	ev, err := NewStatusEvent(
		RecoverableError,
		WithComponentKind(KindExporter),
		WithComponentID(expectedComponentID),
		WithTimestamp(expectedTimestamp),
		WithError(expectedErr),
	)
	assert.NoError(t, err)
	assert.Equal(t, RecoverableError, ev.Type())
	assert.Equal(t, KindExporter, ev.ComponentKind())
	assert.Equal(t, expectedComponentID, ev.ComponentID())
	assert.Equal(t, expectedTimestamp, ev.Timestamp())
	assert.Equal(t, expectedErr, ev.Err())
}

func TestNewEventErrorTypeMismatch(t *testing.T) {
	testCases := []struct {
		name        string
		eventType   StatusEventType
		err         error
		expectError bool
	}{
		{
			"eventType: OK with error returns error",
			OK,
			errors.New("err"),
			true,
		},
		{
			"eventType: RecoverableError with error returns event",
			RecoverableError,
			errors.New("err"),
			false,
		},
		{
			"eventType: PermanentError with error returns event",
			PermanentError,
			errors.New("err"),
			false,
		},
		{
			"eventType: FatalError with error returns event",
			FatalError,
			errors.New("err"),
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ev, err := NewStatusEvent(tc.eventType, WithError(tc.err))
			if tc.expectError {
				assert.Nil(t, ev)
				assert.Error(t, err)
			} else {
				assert.NotNil(t, ev)
				assert.NoError(t, err)
			}
		})
	}
}
