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

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config"
)

func TestHealth_NotificationsSubscribe(t *testing.T) {
	notifications := NewHealthNotifications()

	expectedEvent := HealthEvent{
		ComponentID: config.NewComponentID("nop"),
		Error:       errors.New("an error"),
	}

	sub1 := notifications.Subscribe()
	sub2 := notifications.Subscribe()

	sub1Events := make(chan HealthEvent, 2)
	sub2Events := make(chan HealthEvent, 2)

	sub1Done := make(chan struct{})
	sub2Done := make(chan struct{})

	// consume events on sub1
	go func() {
		for {
			event, ok := <-sub1
			if !ok {
				close(sub1Done)
				return
			}
			sub1Events <- event
		}
	}()

	// consume events on sub2
	go func() {
		for {
			event, ok := <-sub2
			if !ok {
				close(sub2Done)
				return
			}
			sub2Events <- event

		}
	}()

	notifications.Report(expectedEvent)
	notifications.Stop()

	// wait for events to be consumed
	<-sub1Done
	<-sub2Done

	require.Equal(t, 1, len(sub1Events))
	require.Equal(t, 1, len(sub2Events))
	require.Equal(t, expectedEvent, <-sub1Events)
	require.Equal(t, expectedEvent, <-sub2Events)
}

func TestHealth_NotificationsUnsubscribe(t *testing.T) {
	notifications := NewHealthNotifications()

	event1 := HealthEvent{
		ComponentID: config.NewComponentID("nop"),
		Error:       errors.New("an error"),
	}

	event2 := HealthEvent{
		ComponentID: config.NewComponentID("nop"),
		Error:       errors.New("a different error"),
	}

	sub1 := notifications.Subscribe()
	sub2 := notifications.Subscribe()

	sub1Events := make(chan HealthEvent, 2)
	sub2Events := make(chan HealthEvent, 2)

	sub1Done := make(chan struct{})
	sub2Done := make(chan struct{})

	// consume events on sub1
	go func() {
		for {
			event, ok := <-sub1
			if !ok {
				close(sub1Done)
				return
			}
			sub1Events <- event
		}
	}()

	// consume events on sub2
	go func() {
		for {
			event, ok := <-sub2
			if !ok {
				close(sub2Done)
				return
			}
			sub2Events <- event

		}
	}()

	notifications.Report(event1)
	notifications.Unsubscribe(sub2)
	notifications.Report(event2)
	notifications.Stop()

	// wait for events to be consumed
	<-sub1Done
	<-sub2Done

	require.Equal(t, 2, len(sub1Events))
	require.Equal(t, 1, len(sub2Events))
	require.Equal(t, event1, <-sub1Events)
	require.Equal(t, event2, <-sub1Events)
	require.Equal(t, event1, <-sub2Events)
}
