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

// Package client contains generic representations of clients connecting to
// different receivers
package client

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewContext(t *testing.T) {
	testCases := []struct {
		desc string
		cl   Info
	}{
		{
			desc: "valid client",
			cl: Info{
				Addr: &net.IPAddr{
					IP: net.IPv4(1, 2, 3, 4),
				},
			},
		},
		{
			desc: "nil client",
			cl:   Info{},
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
		expected Info
	}{
		{
			desc: "context with client",
			input: context.WithValue(context.Background(), ctxKey{}, Info{
				Addr: &net.IPAddr{
					IP: net.IPv4(1, 2, 3, 4),
				},
			}),
			expected: Info{
				Addr: &net.IPAddr{
					IP: net.IPv4(1, 2, 3, 4),
				},
			},
		},
		{
			desc:     "context without client",
			input:    context.Background(),
			expected: Info{},
		},
		{
			desc:     "context with something else in the key",
			input:    context.WithValue(context.Background(), ctxKey{}, "unexpected!"),
			expected: Info{},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			assert.Equal(t, FromContext(tC.input), tC.expected)
		})
	}
}

func TestMetadata(t *testing.T) {
	source := map[string][]string{"test-key": {"test-val"}}
	md := NewMetadata(source)
	assert.Equal(t, []string{"test-val"}, md.Get("test-key"))
	assert.Equal(t, []string{"test-val"}, md.Get("test-KEY")) // case insensitive lookup

	// test if copy. In regular use, source cannot change
	val := md.Get("test-key")
	source["test-key"][0] = "abc"
	assert.Equal(t, []string{"test-val"}, val)

	assert.Empty(t, md.Get("non-existent-key"))
}
