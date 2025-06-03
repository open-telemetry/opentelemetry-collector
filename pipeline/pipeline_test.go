// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pipeline

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pipeline/internal/globalsignal"
)

func Test_NewID(t *testing.T) {
	id := NewID(SignalTraces)
	assert.Equal(t, ID{signal: SignalTraces}, id)
}

func Test_NewIDWithName(t *testing.T) {
	id := NewIDWithName(SignalTraces, "test")
	assert.Equal(t, ID{signal: SignalTraces, name: "test"}, id)
}

func TestMarshalText(t *testing.T) {
	id := NewIDWithName(SignalTraces, "name")
	got, err := id.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, id.String(), string(got))
}

func TestUnmarshalText(t *testing.T) {
	testCases := []struct {
		idStr       string
		expectedErr bool
		expectedID  ID
	}{
		{
			idStr:      "traces",
			expectedID: ID{signal: globalsignal.SignalTraces, name: ""},
		},
		{
			idStr:      "traces/valid_name",
			expectedID: ID{signal: globalsignal.SignalTraces, name: "valid_name"},
		},
		{
			idStr:      "   traces   /   valid_name  ",
			expectedID: ID{signal: globalsignal.SignalTraces, name: "valid_name"},
		},
		{
			idStr:      "traces/中文好",
			expectedID: ID{signal: globalsignal.SignalTraces, name: "中文好"},
		},
		{
			idStr:      "traces/name-with-dashes",
			expectedID: ID{signal: globalsignal.SignalTraces, name: "name-with-dashes"},
		},
		// issue 10816
		{
			idStr:      "traces/Linux-Messages-File_01J49HCH3SWFXRVASWFZFRT3J2__processor0__logs",
			expectedID: ID{signal: globalsignal.SignalTraces, name: "Linux-Messages-File_01J49HCH3SWFXRVASWFZFRT3J2__processor0__logs"},
		},
		{
			idStr:      "traces/1",
			expectedID: ID{signal: globalsignal.SignalTraces, name: "1"},
		},
		{
			idStr:       "/valid_name",
			expectedErr: true,
		},
		{
			idStr:       "     /valid_name",
			expectedErr: true,
		},
		{
			idStr:       "traces/",
			expectedErr: true,
		},
		{
			idStr:       "traces/      ",
			expectedErr: true,
		},
		{
			idStr:       "      ",
			expectedErr: true,
		},
		{
			idStr:       "traces/invalid name",
			expectedErr: true,
		},
		{
			idStr:       "traces/" + strings.Repeat("a", 1025),
			expectedErr: true,
		},
		{
			idStr:       "INVALID",
			expectedErr: true,
		},
		{
			idStr:       "INVALID/name",
			expectedErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.idStr, func(t *testing.T) {
			id := ID{}
			err := id.UnmarshalText([]byte(test.idStr))
			if test.expectedErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectedID, id)
			assert.Equal(t, test.expectedID.Signal(), id.Signal())
			assert.Equal(t, test.expectedID.Name(), id.Name())
			assert.Equal(t, test.expectedID.String(), id.String())
		})
	}
}
