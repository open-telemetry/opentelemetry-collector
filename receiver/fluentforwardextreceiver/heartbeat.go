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
	"syscall"

	"go.uber.org/zap"
)

// See https://github.com/fluent/fluentd/wiki/Forward-Protocol-Specification-v1#heartbeat-message
func respondToHeartbeats(ctx context.Context, udpSock net.PacketConn, logger *zap.Logger) {
	go func() {
		<-ctx.Done()
		udpSock.Close()
	}()

	buf := make([]byte, 1)
	for {
		n, addr, err := udpSock.ReadFrom(buf)
		if err != nil || n == 0 {
			if ctx.Err() != nil || err == syscall.EINVAL {
				return
			}
			continue
		}
		// Technically the heartbeat should be a byte 0x00 but just echo back
		// whatever the client sent and move on.
		_, err = udpSock.WriteTo(buf, addr)
		if err != nil {
			logger.Debug("Failed to write back heartbeat packet", zap.String("addr", addr.String()), zap.Error(err))
		}
	}
}
