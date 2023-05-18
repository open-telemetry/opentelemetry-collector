// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"errors"
	"testing"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component/componenttest"
)

func Test_newHostWrapper(_ *testing.T) {
	hw := NewHostWrapper(componenttest.NewNopHost(), zap.NewNop())
	hw.ReportFatalError(errors.New("test error"))
}
