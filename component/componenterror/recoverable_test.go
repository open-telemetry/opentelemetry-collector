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

package componenterror

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRecoverable(t *testing.T) {
	var err error
	assert.False(t, IsRecoverable(err))

	err = errors.New("testError")
	assert.False(t, IsRecoverable(err))

	err = NewRecoverable(err)
	assert.True(t, IsRecoverable(err))

	err = fmt.Errorf("%w", err)
	assert.True(t, IsRecoverable(err))
}

func TestRecoverable_Unwrap(t *testing.T) {
	var err error = testErrorType{"testError"}
	require.False(t, IsRecoverable(err))

	// Wrapping testErrorType err with recoverable error.
	recoverableErr := NewRecoverable(err)
	require.True(t, IsRecoverable(recoverableErr))

	target := testErrorType{}
	require.NotEqual(t, err, target)

	isTestErrorTypeWrapped := errors.As(recoverableErr, &target)
	require.True(t, isTestErrorTypeWrapped)

	require.Equal(t, err, target)
}
