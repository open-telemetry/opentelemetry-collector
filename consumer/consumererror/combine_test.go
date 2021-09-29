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
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCombine(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		errors            []error
		expected          string
		expectNil         bool
		expectedPermanent bool
	}{
		{
			errors:    []error{},
			expectNil: true,
		},
		{
			errors: []error{
				fmt.Errorf("foo"),
			},
			expected: "foo",
		},
		{
			errors: []error{
				fmt.Errorf("foo"),
				fmt.Errorf("bar"),
			},
			expected: "[foo; bar]",
		},
		{
			errors: []error{
				fmt.Errorf("foo"),
				fmt.Errorf("bar"),
				NewPermanent(fmt.Errorf("permanent"))},
			expected: "[foo; bar; Permanent error: permanent]",
		},
		{
			errors: []error{
				errors.New("foo"),
				errors.New("bar"),
				nil,
			},
			expected: "[foo; bar]",
		},
		{
			errors: []error{
				errors.New("foo"),
				errors.New("bar"),
				Combine([]error{
					errors.New("foo"),
					errors.New("bar"),
				}),
			},
			expected: "[foo; bar; [foo; bar]]",
		},
	}

	for _, tc := range testCases {
		got := Combine(tc.errors)
		if (got == nil) != tc.expectNil {
			t.Errorf("Combine(%v) == nil? Got: %t. Want: %t", tc.errors, got == nil, tc.expectNil)
		}
		if got != nil && tc.expected != got.Error() {
			t.Errorf("Combine(%v) = %q. Want: %q", tc.errors, got, tc.expected)
		}
		if tc.expectedPermanent && !IsPermanent(got) {
			t.Errorf("Combine(%v) = %q. Want: consumererror.permanent", tc.errors, got)
		}
	}
}

func TestCombineContainsError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		contains error
		err      error
	}{
		{contains: io.EOF, err: Combine([]error{errors.New("invalid entry"), io.EOF})},
		{contains: io.EOF, err: Combine([]error{errors.New("invalid entry"), NewPermanent(io.EOF)})},
		{contains: io.EOF, err: Combine([]error{errors.New("foo"), errors.New("bar"), Combine([]error{errors.New("xyz"), io.EOF})})},
	}

	for _, tc := range testCases {
		assert.ErrorIs(t, tc.err, tc.contains, "Must have the expected error")
	}

	assert.NotErrorIs(t, Combine([]error{errors.New("invalid"), errors.New("foo")}), io.EOF, "Must return false when error does not exist within error")
}
