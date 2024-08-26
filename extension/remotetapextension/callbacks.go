// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package remotetapextension // import "go.opentelemetry.io/collector/extension/remotetapextension"

import (
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/pdata"
)

// CallbackManager registers components and callbacks that can be used to send data to observers.
type CallbackManager struct {
	mu        sync.RWMutex
	callbacks map[pdata.ComponentID]map[CallbackID]func(string)
}

func NewCallbackManager() *CallbackManager {
	return &CallbackManager{
		callbacks: make(map[pdata.ComponentID]map[CallbackID]func(string)),
	}
}

// Add adds a new callback for a registered component.
func (cm *CallbackManager) Add(componentID pdata.ComponentID, callbackID CallbackID, callback func(string)) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, ok := cm.callbacks[componentID]; !ok {
		return fmt.Errorf("componentID %q is not registered to the remoteTap extension", componentID)
	}

	cm.callbacks[componentID][callbackID] = callback
	return nil
}

// Delete deletes a new callback for a registered component.
func (cm *CallbackManager) Delete(componentID pdata.ComponentID, callbackID CallbackID) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.callbacks[componentID], callbackID)
}

// IsActive returns true is the given component has a least one active observer.
func (cm *CallbackManager) IsActive(componentID pdata.ComponentID) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if callbacks, exist := cm.callbacks[componentID]; exist {
		return len(callbacks) > 0
	}

	cm.callbacks[componentID] = make(map[CallbackID]func(string))
	return false
}

// Broadcast sends data from a registered component to its observers.
func (cm *CallbackManager) Broadcast(componentID pdata.ComponentID, data string) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	for _, callback := range cm.callbacks[componentID] {
		callback(data)
	}
}

// GetRegisteredComponents returns all registered components.
func (cm *CallbackManager) GetRegisteredComponents() []pdata.ComponentID {
	components := make([]pdata.ComponentID, 0, len(cm.callbacks))
	for componentID := range cm.callbacks {
		components = append(components, componentID)
	}
	return components
}
