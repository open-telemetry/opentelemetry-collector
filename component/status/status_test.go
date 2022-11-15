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

	"go.opentelemetry.io/collector/component/id"
)

type statusReportingComponent struct{}

func (s statusReportingComponent) ID() id.ID {
	return id.ID{}
}

func TestNewEventMinimal(t *testing.T) {
	ev, err := NewComponentEvent(ComponentOK)
	assert.NoError(t, err)
	assert.Equal(t, ComponentOK, ev.Type())
	assert.NotEqual(t, 0, ev.Timestamp())
	assert.Equal(t, nil, ev.Source())
	assert.Nil(t, ev.Err())
}

func TestNewEventAllOptions(t *testing.T) {
	expectedTimestamp := time.Now()
	expectedErr := errors.New("expect this")
	source := &statusReportingComponent{}
	ev, err := NewComponentEvent(
		ComponentError,
		WithSource(source),
		WithTimestamp(expectedTimestamp),
		WithError(expectedErr),
	)
	assert.NoError(t, err)
	assert.Equal(t, ComponentError, ev.Type())
	assert.Equal(t, source, ev.Source())
	assert.Equal(t, expectedTimestamp, ev.Timestamp())
	assert.Equal(t, expectedErr, ev.Err())
}

func TestNewEventErrorTypeMismatch(t *testing.T) {
	testCases := []struct {
		name        string
		eventType   ComponentEventType
		err         error
		expectError bool
	}{
		{
			"eventType: OK with error returns error",
			ComponentOK,
			errors.New("err"),
			true,
		},
		{
			"eventType: ComponentError with error returns event",
			ComponentError,
			errors.New("err"),
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ev, err := NewComponentEvent(tc.eventType, WithError(tc.err))
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
