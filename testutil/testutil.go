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

package testutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type portpair struct {
	first string
	last  string
}

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

func TempSocketName(t *testing.T) string {
	tmpfile, err := ioutil.TempFile("", "sock")
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())
	socket := tmpfile.Name()
	require.NoError(t, os.Remove(socket))
	return socket
}

// GetAvailableLocalAddress finds an available local port and returns an endpoint
// describing it. The port is available for opening when this function returns
// provided that there is no race by some other code to grab the same port
// immediately.
func GetAvailableLocalAddress(t *testing.T) string {
	ln, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err, "Failed to get a free local port")
	// There is a possible race if something else takes this same port before
	// the test uses it, however, that is unlikely in practice.
	defer ln.Close()
	return ln.Addr().String()
}

// GetAvailablePort finds an available local port and returns it. The port is
// available for opening when this function returns provided that there is no
// race by some other code to grab the same port immediately.
func GetAvailablePort(t *testing.T) uint16 {
	// Retry has been added for windows as net.Listen can return a port that is not actually available. Details can be
	// found in https://github.com/docker/for-win/issues/3171 but to summarize Hyper-V will reserve ranges of ports
	// which do not show up under the "netstat -ano" but can only be found by
	// "netsh interface ipv4 show excludedportrange protocol=tcp".  We'll use []exclusions to hold those ranges and
	// retry if the port returned by GetAvailableLocalAddress falls in one of those them.
	var exclusions []portpair
	portFound := false
	var port string
	var err error
	if runtime.GOOS == "windows" {
		exclusions = getExclusionsList(t)
	}

	for !portFound {
		endpoint := GetAvailableLocalAddress(t)
		_, port, err = net.SplitHostPort(endpoint)
		require.NoError(t, err)
		portFound = true
		if runtime.GOOS == "windows" {
			for _, pair := range exclusions {
				if port >= pair.first && port <= pair.last {
					portFound = false
					break
				}
			}
		}
	}

	portInt, err := strconv.Atoi(port)
	require.NoError(t, err)

	return uint16(portInt)
}

// Get excluded ports on Windows from the command: netsh interface ipv4 show excludedportrange protocol=tcp
func getExclusionsList(t *testing.T) []portpair {
	cmd := exec.Command("netsh", "interface", "ipv4", "show", "excludedportrange", "protocol=tcp")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err)

	exclusions := createExclusionsList(string(output), t)
	return exclusions
}

func createExclusionsList(exclusionsText string, t *testing.T) []portpair {
	exclusions := []portpair{}

	parts := strings.Split(exclusionsText, "--------")
	require.Equal(t, len(parts), 3)
	portsText := strings.Split(parts[2], "*")
	require.Equal(t, len(portsText), 2)
	lines := strings.Split(portsText[0], "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			entries := strings.Fields(strings.TrimSpace(line))
			require.Equal(t, len(entries), 2)
			pair := portpair{entries[0], entries[1]}
			exclusions = append(exclusions, pair)
		}
	}
	return exclusions
}

// WaitForPort repeatedly attempts to open a local port until it either succeeds or 5 seconds pass
// It is useful if you need to asynchronously start a service and wait for it to start
func WaitForPort(t *testing.T, port uint16) error {
	t.Helper()

	totalDuration := 5 * time.Second
	wait := 100 * time.Millisecond
	address := fmt.Sprintf("localhost:%d", port)

	ticker := time.NewTicker(wait)
	defer ticker.Stop()

	timeout := time.After(totalDuration)

	for {
		select {
		case <-ticker.C:
			conn, err := net.Dial("tcp", address)
			if err == nil && conn != nil {
				conn.Close()
				return nil
			}

		case <-timeout:
			return fmt.Errorf("failed to wait for port %d", port)
		}
	}
}

// HostPortFromAddr extracts host and port from a network address
func HostPortFromAddr(addr net.Addr) (host string, port int, err error) {
	addrStr := addr.String()
	sepIndex := strings.LastIndex(addrStr, ":")
	if sepIndex < 0 {
		return "", -1, errors.New("failed to parse host:port")
	}
	host, portStr := addrStr[:sepIndex], addrStr[sepIndex+1:]
	port, err = strconv.Atoi(portStr)
	return host, port, err
}

// WaitFor the specific condition for up to 10 seconds. Records a test error
// if condition does not become true.
func WaitFor(t *testing.T, cond func() bool, errMsg ...interface{}) bool {
	t.Helper()

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

// LimitedWriter is an io.Writer that will return an EOF error after MaxLen has
// been reached.  If MaxLen is 0, Writes will always succeed.
type LimitedWriter struct {
	bytes.Buffer
	MaxLen int
}

var _ io.Writer = new(LimitedWriter)

func (lw *LimitedWriter) Write(p []byte) (n int, err error) {
	if lw.MaxLen != 0 && len(p)+lw.Len() > lw.MaxLen {
		return 0, io.EOF
	}
	return lw.Buffer.Write(p)
}

func (lw *LimitedWriter) Close() error {
	return nil
}
