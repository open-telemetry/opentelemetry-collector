// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMarshalUnmarshalProfilesRequest(t *testing.T) {
	profiles := testdata.GenerateProfiles(3)

	// unmarshal profiles request with a context
	spanCtx := fakeSpanContext(t)
	buf, err := MarshalProfiles(trace.ContextWithSpanContext(context.Background(), spanCtx), profiles)
	require.NoError(t, err)
	gotCtx, gotProfiles, err := UnmarshalProfiles(buf)
	require.NoError(t, err)
	assert.Equal(t, spanCtx, trace.SpanContextFromContext(gotCtx))
	assert.Equal(t, profiles, gotProfiles)

	// unmarshal profiles request with empty context
	buf, err = MarshalProfiles(context.Background(), profiles)
	require.NoError(t, err)
	gotCtx, gotProfiles, err = UnmarshalProfiles(buf)
	require.NoError(t, err)
	assert.Equal(t, context.Background(), gotCtx)
	assert.Equal(t, profiles, gotProfiles)

	// unmarshal corrupted data
	_, _, err = UnmarshalProfiles(buf[:len(buf)-1])
	require.ErrorContains(t, err, "failed to unmarshal profiles request")

	// unmarshal invalid format (bare profiles)
	buf, err = (&pprofile.ProtoMarshaler{}).MarshalProfiles(profiles)
	require.NoError(t, err)
	_, _, err = UnmarshalProfiles(buf)
	require.ErrorIs(t, err, ErrInvalidFormat)
}
