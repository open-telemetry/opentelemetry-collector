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

package status // import "go.opentelemetry.io/collector/component/status"

import (
	"errors"
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/config"
	"go.uber.org/multierr"
)

var errRegistrationNotFound = errors.New("registration not found")

type Notifications struct {
	mu        sync.RWMutex
	listeners []*listener
}

func NewNotifications() *Notifications {
	return &Notifications{
		listeners: []*listener{},
	}
}

func (n *Notifications) Start() error {
	return nil
}

func (n *Notifications) Shutdown() error {
	n.listeners = nil
	return nil
}

func (n *Notifications) RegisterListener(options ...ListenerOption) UnregisterFunc {
	l := newListener(options...)

	n.mu.Lock()
	n.listeners = append(n.listeners, l)
	n.mu.Unlock()

	return func() error {
		return n.unregisterListener(l)
	}
}

func (n *Notifications) ReportStatus(eventType EventType, componentID config.ComponentID, options ...StatusEventOption) (errs error) {
	ev := newStatusEvent(eventType, componentID, options...)

	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, listener := range n.listeners {
		if err := listener.statusEventHandler(ev); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("failure in status event handler: %w", err))
		}

	}

	return errs
}

func (n *Notifications) PipelineReady() (errs error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, listener := range n.listeners {
		if err := listener.pipelineReadyHandler(); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("failure in pipeline ready handler: %w", err))
		}
	}

	return errs
}

func (n *Notifications) PipelineNotReady() (errs error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, listener := range n.listeners {
		if err := listener.pipelineNotReadyHandler(); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("failure in pipeline not ready handler: %w", err))
		}
	}

	return errs
}

func (n *Notifications) unregisterListener(l *listener) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	i := 0
	for _, listener := range n.listeners {
		if l != listener {
			n.listeners[i] = listener
			i++
		}
	}

	if i == len(n.listeners) {
		return errRegistrationNotFound
	}

	for j := i; j < len(n.listeners); j++ {
		n.listeners[j] = nil
	}

	n.listeners = n.listeners[:i]

	return nil
}
