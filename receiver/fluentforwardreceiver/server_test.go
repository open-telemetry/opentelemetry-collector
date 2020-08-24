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

package fluentforwardreceiver

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tinylib/msgp/msgp"
)

func TestDetermineNextEventMode(t *testing.T) {
	cases := []struct {
		name          string
		event         func() []byte
		expectedMode  EventMode
		expectedError error
	}{
		{
			"basic",
			func() []byte {
				var b []byte

				b = msgp.AppendArrayHeader(b, 3)
				b = msgp.AppendString(b, "my-tag")
				b = msgp.AppendInt(b, 5000)
				return b
			},
			MessageMode,
			nil,
		},
		{
			"str8-tag",
			func() []byte {
				var b []byte

				b = msgp.AppendArrayHeader(b, 3)
				b = msgp.AppendString(b, strings.Repeat("a", 128))
				b = msgp.AppendInt(b, 5000)
				return b
			},
			MessageMode,
			nil,
		},
		{
			"str16-tag",
			func() []byte {
				var b []byte

				b = msgp.AppendArrayHeader(b, 3)
				b = msgp.AppendString(b, strings.Repeat("a", 1024))
				b = msgp.AppendInt(b, 5000)
				return b
			},
			MessageMode,
			nil,
		},
		{
			"str32-tag",
			func() []byte {
				var b []byte

				b = msgp.AppendArrayHeader(b, 3)
				b = msgp.AppendString(b, strings.Repeat("a", 66000))
				b = msgp.AppendInt(b, 5000)
				return b
			},
			MessageMode,
			nil,
		},
		{
			"non-string-tag",
			func() []byte {
				var b []byte

				b = msgp.AppendArrayHeader(b, 3)
				b = msgp.AppendInt(b, 10)
				b = msgp.AppendInt(b, 5000)
				return b
			},
			UnknownMode,
			errors.New("malformed tag field"),
		},
		{
			"float-second-elm",
			func() []byte {
				var b []byte

				b = msgp.AppendArrayHeader(b, 3)
				b = msgp.AppendString(b, "my-tag")
				b = msgp.AppendFloat64(b, 5000.0)
				return b
			},
			UnknownMode,
			errors.New("unable to determine next event mode for type float64"),
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			peeker := bufio.NewReaderSize(bytes.NewReader(c.event()), 1024*100)
			mode, err := DetermineNextEventMode(peeker)
			if c.expectedError != nil {
				require.Equal(t, c.expectedError, err)
			} else {
				require.Equal(t, c.expectedMode, mode)
			}
		})
	}
}
