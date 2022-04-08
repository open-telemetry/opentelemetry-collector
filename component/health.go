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

import "go.opentelemetry.io/collector/config"

type HealthEvent struct {
	ComponentID config.ComponentID
	Error       error
}

type HealthEventFunc func(event HealthEvent)

type HealthNotifications struct {
	reporters    []HealthEventFunc
	registerChan chan (HealthEventFunc)
	eventChan    chan (HealthEvent)
	stopChan     chan (struct{})
}

func NewHealthNotifications() *HealthNotifications {
	return &HealthNotifications{
		reporters:    []HealthEventFunc{},
		registerChan: make(chan HealthEventFunc),
		eventChan:    make(chan HealthEvent),
		stopChan:     make(chan struct{}),
	}
}

func (hn *HealthNotifications) Register(reporter HealthEventFunc) {
	hn.registerChan <- reporter
}

func (hn *HealthNotifications) Report(event HealthEvent) {
	hn.eventChan <- event
}

func (hn *HealthNotifications) Start() {
	go func() {
		for {
			select {
			case reporter := <-hn.registerChan:
				hn.reporters = append(hn.reporters, reporter)
			case event := <-hn.eventChan:
				hn.notify(event)
			case <-hn.stopChan:
				return
			}
		}
	}()
}

func (hn *HealthNotifications) Stop() {
	hn.stopChan <- struct{}{}
}

func (hn *HealthNotifications) notify(event HealthEvent) {
	for _, fn := range hn.reporters {
		fn(event)
	}
}
