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

package testbed

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratorAndBackend(t *testing.T) {
	port := GetAvailablePort(t)

	tests := []struct {
		name     string
		receiver DataReceiver
		sender   DataSender
	}{
		{
			name:     "Jaeger-JaegerGRPC",
			receiver: NewJaegerDataReceiver(port),
			sender:   NewJaegerGRPCDataSender(DefaultHost, port),
		},
		{
			name:     "Zipkin-Zipkin",
			receiver: NewZipkinDataReceiver(port),
			sender:   NewZipkinDataSender(DefaultHost, port),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mb := NewMockBackend("mockbackend.log", test.receiver)

			assert.EqualValues(t, 0, mb.DataItemsReceived())
			require.NoError(t, mb.Start(), "Cannot start backend")

			defer mb.Stop()

			options := LoadOptions{DataItemsPerSecond: 10_000, ItemsPerBatch: 10}
			dataProvider := NewPerfTestDataProvider(options)
			lg, err := NewLoadGenerator(dataProvider, test.sender)
			require.NoError(t, err, "Cannot start load generator")

			assert.EqualValues(t, 0, lg.dataItemsSent.Load())

			// Generate at 1000 SPS
			lg.Start(LoadOptions{DataItemsPerSecond: 1000})

			// Wait until at least 50 spans are sent
			WaitFor(t, func() bool { return lg.DataItemsSent() > 50 }, "DataItemsSent > 50")

			lg.Stop()

			// The backend should receive everything generated.
			assert.Equal(t, lg.DataItemsSent(), mb.DataItemsReceived())
		})
	}
}

// WaitFor the specific condition for up to 10 seconds. Records a test error
// if condition does not become true.
func WaitFor(t *testing.T, cond func() bool, errMsg ...interface{}) bool {
	startTime := time.Now()

	// Start with 5 ms waiting interval between condition re-evaluation.
	waitInterval := time.Millisecond * 5

	for {
		time.Sleep(waitInterval)

		// Increase waiting interval exponentially up to 500 ms.
		if waitInterval < time.Millisecond*500 {
			waitInterval *= 2
		}

		if cond() {
			return true
		}

		if time.Since(startTime) > time.Second*10 {
			// Waited too long
			t.Error("Time out waiting for", errMsg)
			return false
		}
	}
}
