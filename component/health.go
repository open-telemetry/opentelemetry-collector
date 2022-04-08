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

package component // import "go.opentelemetry.io/collector/component"

import (
	"sync"

	"go.opentelemetry.io/collector/config"
)

type HealthEvent struct {
	ComponentID config.ComponentID
	Error       error
}

type HealthNotifications struct {
	mu            sync.RWMutex
	subscriptions []chan (HealthEvent)
	eventChan     chan (HealthEvent)
	stopChan      chan (struct{})
}

func NewHealthNotifications() *HealthNotifications {
	return &HealthNotifications{
		subscriptions: []chan (HealthEvent){},
		eventChan:     make(chan HealthEvent),
		stopChan:      make(chan struct{}),
	}
}

func (hn *HealthNotifications) Subscribe() <-chan (HealthEvent) {
	hn.mu.Lock()
	defer hn.mu.Unlock()

	sub := make(chan (HealthEvent))
	hn.subscriptions = append(hn.subscriptions, sub)
	return sub
}

func (hn *HealthNotifications) Report(event HealthEvent) {
	hn.eventChan <- event
}

func (hn *HealthNotifications) Start() {
	go func() {
		for {
			select {
			case event := <-hn.eventChan:
				hn.notify(event)
			case <-hn.stopChan:
				hn.unsubscribeAll()
				return
			}
		}
	}()
}

func (hn *HealthNotifications) Stop() {
	hn.stopChan <- struct{}{}
}

func (hn *HealthNotifications) notify(event HealthEvent) {
	hn.mu.RLock()
	defer hn.mu.RUnlock()

	for _, sub := range hn.subscriptions {
		sub <- event
	}
}

func (hn *HealthNotifications) unsubscribeAll() {
	for i := 0; i < len(hn.subscriptions); i++ {
		close(hn.subscriptions[i])
		hn.subscriptions[i] = nil
	}
	hn.subscriptions = hn.subscriptions[:0]
}
