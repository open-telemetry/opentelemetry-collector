// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"errors"
	"testing"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/internal/servicehost"
)

func Test_newHostWrapper(_ *testing.T) {
	hw := NewHostWrapper(servicehost.NewNopHost(), nil, zap.NewNop())
	hw.ReportFatalError(errors.New("test error"))
	hw.ReportComponentStatus(component.StatusOK)
}
