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

package consumererror

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type testErrorType struct {
	s string
}

func (t testErrorType) Error() string {
	return ""
}

func TestPermanent(t *testing.T) {
	err := errors.New("testError")
	require.False(t, IsPermanent(err))

	err = Permanent(err)
	require.True(t, IsPermanent(err))

	err = fmt.Errorf("%w", err)
	require.True(t, IsPermanent(err))
}

func TestIsPermanent_NilError(t *testing.T) {
	var err error
	require.False(t, IsPermanent(err))
}

func TestPermanent_Unwrap(t *testing.T) {
	var err error = testErrorType{"testError"}
	require.False(t, IsPermanent(err))

	// Wrapping testErrorType err with permanent error.
	permanentErr := Permanent(err)
	require.True(t, IsPermanent(permanentErr))

	target := testErrorType{}
	require.NotEqual(t, err, target)

	isTestErrorTypeWrapped := errors.As(permanentErr, &target)
	require.True(t, isTestErrorTypeWrapped)

	require.Equal(t, err, target)
}
