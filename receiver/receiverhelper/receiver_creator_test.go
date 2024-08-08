// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiverhelper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestReceiverCreator(t *testing.T) {
	settings := receiver.NewSettings(component.MustNewID("foo"), componenttest.NewNopTelemetrySettings(), component.BuildInfo{},
		func(_ component.Type) receiver.Factory {
			return receivertest.NewNopFactory()
		})
	set := NewReceiverCreator(settings)
	f := set.GetReceiverFactory(component.MustNewType("bar"))
	require.Equal(t, component.MustNewType("nop"), f.Type())
}
