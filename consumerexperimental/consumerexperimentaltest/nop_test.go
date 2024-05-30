// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumerexperimentaltest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

func TestNop(t *testing.T) {
	nc := NewNop()
	require.NotNil(t, nc)
	assert.NotPanics(t, nc.unexported)
	assert.NoError(t, nc.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
}
