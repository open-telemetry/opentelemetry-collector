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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimeEventExp(t *testing.T) {
	tim := time.Unix(500, 250)
	e := eventTimeExt(tim)

	require.Equal(t, 8, e.Len())
	var b [8]byte

	err := e.MarshalBinaryTo(b[:])
	require.Nil(t, err)
	require.Equal(t, []byte{0x00, 0x00, 0x01, 0xf4, 0x00, 0x00, 0x00, 0xfa}, b[:])

	err = e.UnmarshalBinary(b[:])
	require.Nil(t, err)
	require.Equal(t, tim, time.Time(e))

	err = e.UnmarshalBinary(b[:5])
	require.NotNil(t, err)
}
