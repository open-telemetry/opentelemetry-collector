// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"errors"
	"testing"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.uber.org/zap"
)

func Test_newHostWrapper(_ *testing.T) {
	hw := NewHostWrapper(componenttest.NewNopHost(), zap.NewNop())
	hw.ReportFatalError(errors.New("test error"))
}
