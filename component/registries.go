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

// Registries struct holds in a single type all registries available for an application.
// Each registry should provide a hook for items to self-register and for consumers to obtain items.
type Registries struct {
	items map[string]*Registry
}

func NewRegistries(items map[string]*Registry) *Registries {
	return &Registries{
		items: items,
	}
}

// Registry holds the instances registered with a particular registry
type Registry []interface{}

// Get returns the registry with the given name, or nil in case none exists under that name
func (r *Registries) Get(registry string) *Registry {
	return r.items[registry]
}

// WantsRegistries defines the interface for components that want to get the list of registries injected.
type WantsRegistries interface {
	SetRegistries(*Registries)
}
