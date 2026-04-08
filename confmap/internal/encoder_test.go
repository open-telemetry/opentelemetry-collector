// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/confmap/internal"

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncoderConfigUnredacted(t *testing.T) {
	opts := MarshalOptions{OpaqueUnredacted: true}
	cfg := EncoderConfig(nil, opts)
	require.NotNil(t, cfg)
}
