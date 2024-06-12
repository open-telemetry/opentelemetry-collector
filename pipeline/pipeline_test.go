// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
)

func TestNewPipelineID(t *testing.T) {
	id := NewID(component.DataTypeMetrics)
	assert.Equal(t, "", id.nameVal)
}

func TestPipelineIDMarshalText(t *testing.T) {
	id := NewIDWithName(component.DataTypeMetrics, "name")
	got, err := id.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, id.String(), string(got))
}

func TestPipelineIDUnmarshalText(t *testing.T) {
	var testCases = []struct {
		idStr       string
		expectedErr bool
		expectedID  ID
	}{
		{
			idStr:      "metrics",
			expectedID: ID{typeVal: component.DataTypeMetrics, nameVal: ""},
		},
		{
			idStr:      "logs/valid_name",
			expectedID: ID{typeVal: component.DataTypeLogs, nameVal: "valid_name"},
		},
		{
			idStr:      "   traces   /   valid_name  ",
			expectedID: ID{typeVal: component.DataTypeTraces, nameVal: "valid_name"},
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
			idStr:       "valid_type/",
			expectedErr: true,
		},
		{
			idStr:       "valid_type/      ",
			expectedErr: true,
		},
		{
			idStr:       "      ",
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

			assert.NoError(t, err)
			assert.Equal(t, test.expectedID, id)
			assert.Equal(t, test.expectedID.Type(), id.Type())
			assert.Equal(t, test.expectedID.Name(), id.Name())
			assert.Equal(t, test.expectedID.String(), id.String())
		})
	}
}
