// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/internal/servicehost"
)

func Test_newHostWrapper(t *testing.T) {
	hw := NewHostWrapper(servicehost.NewNopHost(), nil, zap.NewNop())
	hw.ReportFatalError(errors.New("test error"))
	ev, err := component.NewStatusEvent(component.StatusOK)
	assert.NoError(t, err)
	hw.ReportComponentStatus(ev)
}
