// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/internal/metadata"
)

// CopyTo has always handed the destination the source's own byte slice for plain bytes fields, so
// the two alias afterwards. That is worth keeping with the gate off, where the point is to be
// indistinguishable from not having the feature. With an arena it cannot be kept: the destination
// stores the slice header in memory the garbage collector never scans, so it has to own the bytes.
func TestProfileCopyToSharesPayloadOnlyWithoutArena(t *testing.T) {
	for _, tc := range []struct {
		name       string
		gate       bool
		wantShared bool
	}{
		{"gate off", false, true},
		{"gate on", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prev := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
			require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), tc.gate))
			t.Cleanup(func() {
				require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), prev))
			})

			src := NewProfile()
			src.OriginalPayload().FromRaw([]byte{1, 2, 3})
			dest := NewProfile()
			src.CopyTo(dest)

			require.Equal(t, []byte{1, 2, 3}, dest.OriginalPayload().AsRaw())
			shared := &src.orig.OriginalPayload[0] == &dest.orig.OriginalPayload[0]
			assert.Equal(t, tc.wantShared, shared)

			// Writing through the source is the observable consequence of sharing.
			src.orig.OriginalPayload[0] = 99
			assert.Equal(t, tc.wantShared, dest.OriginalPayload().AsRaw()[0] == 99)
		})
	}
}
