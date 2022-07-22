// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nonfatalerror

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testErrorType struct {
	s string
}

func (t testErrorType) Error() string {
	return ""
}

func TestIsNonFatal(t *testing.T) {
	var err error
	assert.False(t, IsNonFatal(err))

	err = errors.New("testError")
	assert.False(t, IsNonFatal(err))

	err = New(err)
	assert.True(t, IsNonFatal(err))

	err = fmt.Errorf("%w", err)
	assert.True(t, IsNonFatal(err))
}

func TestNonFatal_Unwrap(t *testing.T) {
	var err error = testErrorType{"testError"}
	require.False(t, IsNonFatal(err))

	// Wrapping testErrorType err with non fatal error.
	nonFatalErr := New(err)
	require.True(t, IsNonFatal(nonFatalErr))

	target := testErrorType{}
	require.NotEqual(t, err, target)

	isTestErrorTypeWrapped := errors.As(nonFatalErr, &target)
	require.True(t, isTestErrorTypeWrapped)

	require.Equal(t, err, target)
}
