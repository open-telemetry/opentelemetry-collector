// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumerexperimentaltest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/pprofile/testdata"
)

func TestProfilesSink(t *testing.T) {
	sink := new(ProfilesSink)
	td := testdata.GenerateProfiles(1)
	want := make([]pprofile.Profiles, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeProfiles(context.Background(), td))
		want = append(want, td)
	}
	assert.Equal(t, want, sink.AllProfiles())
	assert.Equal(t, len(want), sink.ProfilesCount())
	sink.Reset()
	assert.Equal(t, 0, len(sink.AllProfiles()))
	assert.Equal(t, 0, sink.ProfilesCount())
}
