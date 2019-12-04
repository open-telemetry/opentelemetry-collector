// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutils

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// GenerateNormalizedJSON generates a normalized JSON from the string
// given to the function. Useful to compare JSON contents that
// may have differences due to formatting. It returns nil in case of
// invalid JSON.
func GenerateNormalizedJSON(t *testing.T, jsonStr string) string {
	var i interface{}

	err := json.Unmarshal([]byte(jsonStr), &i)
	require.NoError(t, err)

	n, err := json.Marshal(i)
	require.NoError(t, err)

	return string(n)
}

// GetAvailableLocalAddress finds an available local port and returns an endpoint
// describing it. The port is available for opening when this function returns
// provided that there is no race by some other code to grab the same port
// immediately.
func GetAvailableLocalAddress(t *testing.T) string {
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to get a free local port: %v", err)
	}
	// There is a possible race if something else takes this same port before
	// the test uses it, however, that is unlikely in practice.
	defer ln.Close()
	return ln.Addr().String()
}

// GetAvailablePort finds an available local port and returns it. The port is
// available for opening when this function returns provided that there is no
// race by some other code to grab the same port immediately.
func GetAvailablePort(t *testing.T) uint16 {
	endpoint := GetAvailableLocalAddress(t)
	_, port, err := net.SplitHostPort(endpoint)
	require.NoError(t, err)

	portInt, err := strconv.Atoi(port)
	require.NoError(t, err)

	return uint16(portInt)
}

// WaitForPort repeatedly attempts to open a local port until it either succeeds or 5 seconds pass
// It is useful if you need to asynchronously start a service and wait for it to start
func WaitForPort(t *testing.T, port uint16) error {
	t.Helper()

	totalDuration := 5 * time.Second
	wait := 100 * time.Millisecond
	address := fmt.Sprintf("localhost:%d", port)
	for i := totalDuration; i > 0; i -= wait {
		conn, err := net.Dial("tcp", address)

		if err == nil && conn != nil {
			conn.Close()
			return nil
		}
		time.Sleep(wait)
	}
	return fmt.Errorf("failed to wait for port %d", port)
}
