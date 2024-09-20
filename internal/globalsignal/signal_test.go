// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package globalsignal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewSignal(t *testing.T) {
	s, err := NewSignal("traces")
	require.NoError(t, err)
	assert.Equal(t, Signal{name: "traces"}, s)
}

func Test_NewSignal_Invalid(t *testing.T) {
	_, err := NewSignal("")
	require.Error(t, err)
	_, err = NewSignal("TRACES")
	require.Error(t, err)
}

func Test_MustNewSignal(t *testing.T) {
	s := MustNewSignal("traces")
	assert.Equal(t, Signal{name: "traces"}, s)
}

func Test_Signal_String(t *testing.T) {
	s := MustNewSignal("traces")
	assert.Equal(t, "traces", s.String())
}

func Test_Signal_MarshalText(t *testing.T) {
	s := MustNewSignal("traces")
	b, err := s.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, []byte("traces"), b)
}
