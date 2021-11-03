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

// Package client contains generic representations of clients connecting to different receivers
package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewContext(t *testing.T) {
	testCases := []struct {
		desc string
		cl   *Client
	}{
		{
			desc: "valid client",
			cl: &Client{
				IP: "1.2.3.4",
			},
		},
		{
			desc: "nil client",
			cl:   nil,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			ctx := NewContext(context.Background(), tC.cl)
			assert.Equal(t, ctx.Value(ctxKey{}), tC.cl)

		})
	}
}

func TestFromContext(t *testing.T) {
	testCases := []struct {
		desc     string
		input    context.Context
		expected *Client
	}{
		{
			desc: "context with client",
			input: context.WithValue(context.Background(), ctxKey{}, &Client{
				IP: "1.2.3.4",
			}),
			expected: &Client{
				IP: "1.2.3.4",
			},
		},
		{
			desc:     "context without client",
			input:    context.Background(),
			expected: &Client{},
		},
		{
			desc:     "context with something else in the key",
			input:    context.WithValue(context.Background(), ctxKey{}, "unexpected!"),
			expected: &Client{},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			assert.Equal(t, FromContext(tC.input), tC.expected)
		})
	}
}
