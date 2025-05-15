// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconsumererror // import "go.opentelemetry.io/collector/consumer/consumererror/xconsumererror"

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestProfiles(t *testing.T) {
	td := testdata.GenerateProfiles(1)
	err := errors.New("some error")
	profileErr := NewProfiles(err, td)
	assert.Equal(t, err.Error(), profileErr.Error())
	var target Profiles
	assert.NotErrorAs(t, nil, &target)
	assert.NotErrorAs(t, err, &target)
	require.ErrorAs(t, profileErr, &target)
	assert.Equal(t, td, target.Data())
}

func TestProfiles_Unwrap(t *testing.T) {
	td := testdata.GenerateProfiles(1)
	var err error = testErrorType{"some error"}
	// Wrapping err with error Profiles.
	profileErr := NewProfiles(err, td)
	target := testErrorType{}
	require.NotEqual(t, err, target)
	// Unwrapping profileErr for err and assigning to target.
	require.ErrorAs(t, profileErr, &target)
	require.Equal(t, err, target)
}

type testErrorType struct {
	s string
}

func (t testErrorType) Error() string {
	return ""
}
