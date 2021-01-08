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
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tinylib/msgp/msgp"

	"go.opentelemetry.io/collector/testutil"
)

func msgpWriterWithLimit(t *testing.T, l int) *msgp.Writer {
	// NewWriterSize forces size to be at least 18 bytes so just use that as
	// the floor and write nulls to those first 18 bytes to make the limit
	// truly l.
	w := msgp.NewWriterSize(&testutil.LimitedWriter{
		MaxLen: l,
	}, 18+l)
	_, err := w.Write(bytes.Repeat([]byte{0x00}, 18))
	require.NoError(t, err)
	return w
}

func TestAckEncoding(t *testing.T) {
	a := &AckResponse{
		Ack: "test",
	}

	err := a.EncodeMsg(msgpWriterWithLimit(t, 1000))
	require.Nil(t, err)

	err = a.EncodeMsg(msgpWriterWithLimit(t, 4))
	require.NotNil(t, err)

	err = a.EncodeMsg(msgpWriterWithLimit(t, 7))
	require.NotNil(t, err)
}
