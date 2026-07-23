// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAvailableLocalAddress(t *testing.T) {
	endpoint := GetAvailableLocalAddress(t)

	lc := &net.ListenConfig{}

	// Endpoint should be free.
	ln0, err := lc.Listen(t.Context(), "tcp", endpoint)
	require.NoError(t, err)
	require.NotNil(t, ln0)
	t.Cleanup(func() {
		assert.NoError(t, ln0.Close())
	})

	// Ensure that the endpoint wasn't something like ":0" by checking that a
	// second listener will fail.
	ln1, err := lc.Listen(t.Context(), "tcp", endpoint)
	require.Error(t, err)
	require.Nil(t, ln1)
}

func TestGetAvailableLocalIpv6Address(t *testing.T) {
	endpoint := GetAvailableLocalIPv6Address(t)

	lc := &net.ListenConfig{}
	// Endpoint should be free.
	ln0, err := lc.Listen(t.Context(), "tcp", endpoint)
	require.NoError(t, err)
	require.NotNil(t, ln0)
	t.Cleanup(func() {
		assert.NoError(t, ln0.Close())
	})

	// Ensure that the endpoint wasn't something like ":0" by checking that a
	// second listener will fail.
	ln1, err := lc.Listen(t.Context(), "tcp", endpoint)
	require.Error(t, err)
	require.Nil(t, ln1)
}

func TestCreateExclusionsList(t *testing.T) {
	// Test two examples of typical output from "netsh interface ipv4 show excludedportrange protocol=tcp"
	emptyExclusionsText := `

Protocol tcp Port Exclusion Ranges

Start Port    End Port
----------    --------

* - Administered port exclusions.`

	exclusionsText := `

Start Port    End Port
----------    --------
     49697       49796
     49797       49896

* - Administered port exclusions.
`
	exclusions := createExclusionsList(t, exclusionsText)
	require.Len(t, exclusions, 2)

	emptyExclusions := createExclusionsList(t, emptyExclusionsText)
	require.Empty(t, emptyExclusions)
}

func TestGetExclusionsListWithMockedNetsh(t *testing.T) {
	mockDir := t.TempDir()
	mockNetsh := filepath.Join(mockDir, "netsh")

	mockOutput := `

Start Port    End Port
----------    --------
     49697       49796
     49797       49896

* - Administered port exclusions.
`

	if runtime.GOOS == "windows" {
		mockNetsh += ".bat"
		batch := "@echo off\r\n" +
			"echo.\r\n" +
			"echo Start Port    End Port\r\n" +
			"echo ----------    --------\r\n" +
			"echo      49697       49796\r\n" +
			"echo      49797       49896\r\n" +
			"echo.\r\n" +
			"echo * - Administered port exclusions.\r\n"
		require.NoError(t, os.WriteFile(mockNetsh, []byte(batch), 0o700)) // #nosec G306 -- test helper executable
	} else {
		require.NoError(t, os.WriteFile(mockNetsh, []byte("#!/bin/sh\nprintf '%s' \""+mockOutput+"\"\n"), 0o700)) // #nosec G306 -- test helper executable
	}
	t.Setenv("PATH", mockDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	t.Run("tcp", func(t *testing.T) {
		exclusions := getExclusionsList(t, "tcp")
		require.Len(t, exclusions, 4)
	})

	t.Run("tcp6", func(t *testing.T) {
		exclusions := getExclusionsList(t, "tcp6")
		require.Len(t, exclusions, 4)
	})
}
