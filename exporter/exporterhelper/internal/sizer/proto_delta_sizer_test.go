// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sizer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetricsBytesDeltaSize(t *testing.T) {
	sizer := protoDeltaSizer{}
	require.Equal(t, 129, sizer.DeltaSize(127))
	require.Equal(t, 131, sizer.DeltaSize(128))
	require.Equal(t, 242, sizer.DeltaSize(239))
}
