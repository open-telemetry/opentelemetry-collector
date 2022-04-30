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

	"go.uber.org/multierr"
)

var errRegistrationNotFound = errors.New("registration not found")

// Notifications is a mechanism that facilitates the passing of status events between components.
// Status events can be used to indicate that a component is functioning as expected, or that it is
// in an error state. Components can send status events using the ReportStatus function, and
// components interested in receiving events can register a listener via the RegisterListener
// function.
type Notifications struct {
	mu        sync.RWMutex
	listeners []*listener
}

// NewNotifications returns a pointer to a newly initialized Notifications struct
func NewNotifications() *Notifications {
	return &Notifications{
		listeners: []*listener{},
	}
}

// Start starts notifications
func (n *Notifications) Start() error {
	return nil
}

// Shutdown stops notifications
func (n *Notifications) Shutdown() error {
	n.listeners = nil
	return nil
}

// RegisterListener registers a component to listen to the events indicated by the passed in
// ListenerOptions. It returns an UnregisterFunc, which should be called by the registering
// component during Shutdown.
func (n *Notifications) RegisterListener(options ...ListenerOption) UnregisterFunc {
	l := newListener(options...)

	n.mu.Lock()
	n.listeners = append(n.listeners, l)
	n.mu.Unlock()

	return func() error {
		return n.unregisterListener(l)
	}
}

// ReportStatus notifies all registered listeners of a StatusEvent. A StatusEvent can indicate that
// a component is functioning normally (status.EventOK) or is in an error state (status.EventError)
func (n *Notifications) ReportStatus(event StatusEvent) (errs error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, listener := range n.listeners {
		if err := listener.statusEventHandler(event); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("failure in status event handler: %w", err))
		}

	}

	return errs
}

// PipelineReady is to be called by the service when all pipelines have been built and the receivers
// have started, i.e.: the service is ready to receive data (note that it may already have received
// data when this method is called). All listeners that have registered a
// PipelineReadyHandler will be notified.
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

// PipelineNotReady is to be called by the service when all receivers are about to be stopped,
// i.e.: pipeline receivers will not accept new data. All listeners with a registered
// PipelineNotReady handler will be notified. The notification is sent before receivers are stopped,
// so the Extension can take any appropriate actions before that happens.
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
