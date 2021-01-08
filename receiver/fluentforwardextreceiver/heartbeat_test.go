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
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUDPHeartbeat(t *testing.T) {
	udpSock, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go respondToHeartbeats(ctx, udpSock, zap.NewNop())

	conn, err := net.Dial("udp", udpSock.LocalAddr().String())
	require.Nil(t, err)

	n, err := conn.Write([]byte{0x00})
	require.Nil(t, err)
	require.Equal(t, 1, n)

	buf := make([]byte, 1)
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
	n, err = conn.Read(buf)
	require.Nil(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, uint8(0x00), buf[0])
}
