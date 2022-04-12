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

package service // import "go.opentelemetry.io/collector/service"

import (
	"sync"

	"go.opentelemetry.io/collector/component"
)

type healthNotifications struct {
	mu            sync.RWMutex
	subscriptions []chan (component.HealthEvent)
	eventChan     chan (component.HealthEvent)
}

func newHealthNotifications() *healthNotifications {
	return &healthNotifications{
		subscriptions: []chan (component.HealthEvent){},
		eventChan:     make(chan component.HealthEvent),
	}
}

func (hn *healthNotifications) Subscribe() <-chan (component.HealthEvent) {
	hn.mu.Lock()
	defer hn.mu.Unlock()

	sub := make(chan (component.HealthEvent), 1)
	hn.subscriptions = append(hn.subscriptions, sub)
	return sub
}

func (hn *healthNotifications) Unsubscribe(subscription <-chan (component.HealthEvent)) {
	hn.mu.Lock()
	defer hn.mu.Unlock()

	i := 0
	for _, sub := range hn.subscriptions {
		if sub != subscription {
			hn.subscriptions[i] = sub
			i++
		} else {
			close(sub)
		}
	}

	for j := i; j < len(hn.subscriptions); j++ {
		hn.subscriptions[j] = nil
	}

	hn.subscriptions = hn.subscriptions[0:i]
}

func (hn *healthNotifications) Send(event component.HealthEvent) {
	hn.mu.RLock()
	defer hn.mu.RUnlock()

	for _, sub := range hn.subscriptions {
		sub <- event
	}
}

func (hn *healthNotifications) Stop() {
	hn.mu.Lock()
	defer hn.mu.Unlock()

	for i := 0; i < len(hn.subscriptions); i++ {
		close(hn.subscriptions[i])
		hn.subscriptions[i] = nil
	}
	hn.subscriptions = hn.subscriptions[:0]
}
