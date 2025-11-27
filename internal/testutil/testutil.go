// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testutil // import "go.opentelemetry.io/collector/internal/testutil"

import (
	"net"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type portpair struct {
	first string
	last  string
}

// GetAvailableLocalAddress finds an available local port and returns an endpoint
// describing it. The port is available for opening when this function returns
// provided that there is no race by some other code to grab the same port
// immediately.
func GetAvailableLocalAddress(tb testing.TB) string {
	return findAvailable(tb, "tcp4")
}

// GetAvailableLocalIPv6Address is IPv6 version of GetAvailableLocalAddress.
func GetAvailableLocalIPv6Address(tb testing.TB) string {
	return findAvailable(tb, "tcp6")
}

func findAvailable(tb testing.TB, network string) string {
	// Retry has been added for windows as net.Listen can return a port that is not actually available. Details can be
	// found in https://github.com/docker/for-win/issues/3171 but to summarize Hyper-V will reserve ranges of ports
	// which do not show up under the "netstat -ano" but can only be found by
	// "netsh interface ipv4 show excludedportrange protocol=tcp".  We'll use []exclusions to hold those ranges and
	// retry if the port returned by GetAvailableLocalAddress falls in one of those them.
	var exclusions []portpair
	portFound := false
	if runtime.GOOS == "windows" {
		exclusions = getExclusionsList(tb, network)
	}

	var endpoint string
	for !portFound {
		endpoint = findAvailableAddress(tb, network)
		_, port, err := net.SplitHostPort(endpoint)
		require.NoError(tb, err)
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

	return endpoint
}

func findAvailableAddress(tb testing.TB, network string) string {
	var host string
	switch network {
	case "tcp", "tcp4":
		host = "localhost"
	case "tcp6":
		host = "[::1]"
	}
	require.NotEmpty(tb, host, "network must be either of tcp, tcp4 or tcp6")

	ln, err := net.Listen("tcp", host+":0")
	require.NoError(tb, err, "Failed to get a free local port")
	// There is a possible race if something else takes this same port before
	// the test uses it, however, that is unlikely in practice.
	defer func() {
		assert.NoError(tb, ln.Close())
	}()
	return ln.Addr().String()
}

// Get excluded ports on Windows from the command: netsh interface ipv4 show excludedportrange protocol=tcp
func getExclusionsList(tb testing.TB, network string) []portpair {
	var cmdTCP *exec.Cmd
	switch network {
	case "tcp", "tcp4":
		cmdTCP = exec.Command("netsh", "interface", "ipv4", "show", "excludedportrange", "protocol=tcp")
	case "tcp6":
		cmdTCP = exec.Command("netsh", "interface", "ipv6", "show", "excludedportrange", "protocol=tcp")
	}
	require.NotZero(tb, cmdTCP, "network must be either of tcp, tcp4 or tcp6")

	outputTCP, errTCP := cmdTCP.CombinedOutput()
	require.NoError(tb, errTCP)
	exclusions := createExclusionsList(tb, string(outputTCP))

	cmdUDP := exec.Command("netsh", "interface", "ipv4", "show", "excludedportrange", "protocol=udp")
	outputUDP, errUDP := cmdUDP.CombinedOutput()
	require.NoError(tb, errUDP)
	exclusions = append(exclusions, createExclusionsList(tb, string(outputUDP))...)

	return exclusions
}

func createExclusionsList(tb testing.TB, exclusionsText string) []portpair {
	var exclusions []portpair

	parts := strings.Split(exclusionsText, "--------")
	require.Len(tb, parts, 3)
	portsText := strings.Split(parts[2], "*")
	require.Greater(tb, len(portsText), 1) // original text may have a suffix like " - Administered port exclusions."
	lines := strings.SplitSeq(portsText[0], "\n")
	for line := range lines {
		if strings.TrimSpace(line) != "" {
			entries := strings.Fields(strings.TrimSpace(line))
			require.Len(tb, entries, 2)
			pair := portpair{entries[0], entries[1]}
			exclusions = append(exclusions, pair)
		}
	}
	return exclusions
}
