// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconfmap

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap/internal"
)

func TestWithUnredacted(t *testing.T) {
	opt := WithUnredacted()
	require.NotNil(t, opt)

	opts := &internal.MarshalOptions{}
	opt.(internal.MarshalOptionFunc)(opts)
	require.True(t, opts.OpaqueUnredacted)
}
