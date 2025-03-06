// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package oteltest // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/oteltest"

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func CheckStatus(t *testing.T, sd sdktrace.ReadOnlySpan, err error) {
	if err != nil {
		require.Equal(t, codes.Error, sd.Status().Code)
		require.EqualError(t, err, sd.Status().Description)
	} else {
		require.Equal(t, codes.Unset, sd.Status().Code)
	}
}
