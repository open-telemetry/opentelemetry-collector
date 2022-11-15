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

package components // import "go.opentelemetry.io/collector/service/internal/components"

import (
	"errors"
	"fmt"
	"sync"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/status"
)

var errRegistrationNotFound = errors.New("registration not found")

// Notifications is a mechanism that facilitates the passing of status events between components.
// Status events can be used to indicate that a component is functioning as expected, or that it is
// in an error state. Events can be sent using the SetComponentStatus function, and
// parties interested in receiving events can register a listener via the RegisterListener
// function.
// This functionality is exposed to the components via the Host interface.
type Notifications struct {
	mu        sync.RWMutex
	listeners []*status.Listener
}

// NewNotifications returns a pointer to a newly initialized Notifications struct
func NewNotifications() *Notifications {
	return &Notifications{
		listeners: []*status.Listener{},
	}
}

// Shutdown stops notifications
func (n *Notifications) Shutdown() {
	n.mu.Lock()
	n.listeners = nil
	n.mu.Unlock()
}

// RegisterListener registers a component to listen to the events indicated by the passed in
// ListenerOptions. It returns an UnregisterFunc, which should be called by the registering
// component during Shutdown.
func (n *Notifications) RegisterListener(options ...status.ListenerOption) component.StatusListenerUnregisterFunc {
	l := status.NewStatusListener(options...)

	n.mu.Lock()
	n.listeners = append(n.listeners, l)
	n.mu.Unlock()

	return func() error {
		return n.unregisterListener(l)
	}
}

// SetComponentStatus notifies all registered listeners of a ComponentEvent. A ComponentEvent can indicate that
// a component is functioning normally (component.StatusEventOK) or is in an error state (component.StatusEventError)
func (n *Notifications) SetComponentStatus(event *status.ComponentEvent) (errs error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, listener := range n.listeners {
		if err := listener.ComponentEventHandler(event); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("failure in status event handler: %w", err))
		}

	}

	return errs
}

// SetPipelineStatus is to be called by the service when all pipelines have been built and the receivers
// have started, i.e.: the service is ready to receive data (note that it may already have received
// data when this method is called). All listeners that have registered a
// PipelineReadyHandler will be notified.
func (n *Notifications) SetPipelineStatus(status status.PipelineReadiness) (errs error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, listener := range n.listeners {
		if err := listener.PipelineStatusHandler(status); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("failure in pipeline status handler: %w", err))
		}
	}

	return errs
}

func (n *Notifications) unregisterListener(l *status.Listener) error {
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
