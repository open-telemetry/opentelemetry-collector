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

package sharedcomponent

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
)

type SharedComponents struct {
	comps map[interface{}]*SharedComponent
}

func NewSharedComponents() *SharedComponents {
	return &SharedComponents{
		comps: make(map[interface{}]*SharedComponent),
	}
}

func (scs *SharedComponents) GetOrAdd(key interface{}, create func() component.Component) *SharedComponent {
	if c, ok := scs.comps[key]; ok {
		return c
	}
	newComp := &SharedComponent{
		Component: create(),
		removeFunc: func() {
			delete(scs.comps, key)
		},
	}
	scs.comps[key] = newComp
	return newComp
}

type SharedComponent struct {
	component.Component

	startOnce  sync.Once
	stopOnce   sync.Once
	removeFunc func()
}

// Unwrap returns the original component.
func (r *SharedComponent) Unwrap() component.Component {
	return r.Component
}

// Start implements component.Component.
func (r *SharedComponent) Start(ctx context.Context, host component.Host) error {
	var err error
	r.startOnce.Do(func() {
		err = r.Component.Start(ctx, host)
	})
	return err
}

// Shutdown implements component.Component.
func (r *SharedComponent) Shutdown(ctx context.Context) error {
	var err error
	r.stopOnce.Do(func() {
		err = r.Component.Shutdown(ctx)
		r.removeFunc()
	})
	return err
}
