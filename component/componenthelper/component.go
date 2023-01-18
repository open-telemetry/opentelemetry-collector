// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package componenthelper // import "go.opentelemetry.io/collector/component/componenthelper"

import (
	"context"

	"go.uber.org/atomic"

	"go.opentelemetry.io/collector/component"
)

const (
	lifecycleShutdown = iota
	lifecycleRunning
)

type baseComponent struct {
	current    *atomic.Int64
	startFn    component.StartFunc
	shutdownFn component.ShutdownFunc
}

// NewComponent creates a component with the following guarantees
// in order to comply with expectations for component.Component:
//
// - Start will only be called once while shutdown
// - Shutdown will only be called if the component is started
// - Allow the component restart the component lifecycle
func NewComponent(start component.StartFunc, shutdown component.ShutdownFunc) component.Component {
	return &baseComponent{
		current:    atomic.NewInt64(lifecycleShutdown),
		startFn:    start,
		shutdownFn: shutdown,
	}
}

func (bc *baseComponent) Start(ctx context.Context, host component.Host) error {
	if previous := bc.current.Swap(lifecycleRunning); previous != lifecycleShutdown {
		return nil
	}
	return bc.startFn.Start(ctx, host)
}

func (bc *baseComponent) Shutdown(ctx context.Context) error {
	if previous := bc.current.Swap(lifecycleShutdown); previous != lifecycleRunning {
		return nil
	}
	return bc.shutdownFn.Shutdown(ctx)
}
