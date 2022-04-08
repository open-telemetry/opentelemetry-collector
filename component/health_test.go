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
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config"
)

func TestHealth_NotificationsSubscribe(t *testing.T) {
	notifications := NewHealthNotifications()

	expectedEvent := HealthEvent{
		ComponentID: config.NewComponentID("nop"),
		Error:       errors.New("an error"),
	}

	notifications.Start()
	sub1 := notifications.Subscribe()
	sub2 := notifications.Subscribe()

	sub1Events := []HealthEvent{}
	sub2Events := []HealthEvent{}

	go func() {
		for event := range sub1 {
			sub1Events = append(sub1Events, event)
		}
	}()

	go func() {
		for event := range sub2 {
			sub2Events = append(sub2Events, event)
		}
	}()

	notifications.Report(expectedEvent)

	expectedEvents := []HealthEvent{expectedEvent}

	require.Eventually(t, func() bool {
		return assert.Equal(t, expectedEvents, sub1Events) && assert.Equal(t, expectedEvents, sub2Events)
	}, time.Second, time.Microsecond)

	notifications.Stop()
}
