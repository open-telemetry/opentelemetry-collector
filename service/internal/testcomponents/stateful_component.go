// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testcomponents // import "go.opentelemetry.io/collector/service/internal/testcomponents"

import (
	"context"

	"go.opentelemetry.io/collector/component"
)

type componentState struct {
	started bool
	stopped bool
}

func (cs *componentState) Started() bool {
	return cs.started
}

func (cs *componentState) Stopped() bool {
	return cs.stopped
}

func (cs *componentState) Start(context.Context, component.Host) error {
	cs.started = true
	return nil
}

func (cs *componentState) Shutdown(context.Context) error {
	cs.stopped = true
	return nil
}
