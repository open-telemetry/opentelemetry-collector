// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confignet

import (
	"context"
	"errors"
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultDialerConfig(t *testing.T) {
	expectedDialerConfig := DialerConfig{}
	dialerConfig := NewDefaultDialerConfig()
	require.Equal(t, expectedDialerConfig, dialerConfig)
}

func TestNewDefaultAddrConfig(t *testing.T) {
	expectedAddrConfig := AddrConfig{}
	addrConfig := NewDefaultAddrConfig()
	require.Equal(t, expectedAddrConfig, addrConfig)
}

func TestNewDefaultTCPAddrConfig(t *testing.T) {
	expectedTCPAddrConfig := TCPAddrConfig{}
	tcpAddrconfig := NewDefaultTCPAddrConfig()
	require.Equal(t, expectedTCPAddrConfig, tcpAddrconfig)
}

func TestAddrConfigTimeout(t *testing.T) {
	nac := &AddrConfig{
		Endpoint:  "localhost:0",
		Transport: TransportTypeTCP,
		DialerConfig: DialerConfig{
			Timeout: -1 * time.Second,
		},
	}
	_, err := nac.Dial(context.Background())
	require.Error(t, err)
	var netErr net.Error
	if errors.As(err, &netErr) {
		assert.True(t, netErr.Timeout())
	} else {
		assert.Fail(t, "error should be a net.Error")
	}
}

func TestTCPAddrConfigTimeout(t *testing.T) {
	nac := &TCPAddrConfig{
		Endpoint: "localhost:0",
		DialerConfig: DialerConfig{
			Timeout: -1 * time.Second,
		},
	}
	_, err := nac.Dial(context.Background())
	require.Error(t, err)
	var netErr net.Error
	if errors.As(err, &netErr) {
		assert.True(t, netErr.Timeout())
	} else {
		assert.Fail(t, "error should be a net.Error")
	}
}

func TestAddrConfig(t *testing.T) {
	nas := &AddrConfig{
		Endpoint:  "localhost:0",
		Transport: TransportTypeTCP,
	}
	ln, err := nas.Listen(context.Background())
	require.NoError(t, err)
	done := make(chan bool, 1)

	go func() {
		conn, errGo := ln.Accept()
		assert.NoError(t, errGo)
		buf := make([]byte, 10)
		var numChr int
		numChr, errGo = conn.Read(buf)
		assert.NoError(t, errGo)
		assert.Equal(t, "test", string(buf[:numChr]))
		assert.NoError(t, conn.Close())
		done <- true
	}()

	nac := &AddrConfig{
		Endpoint:  ln.Addr().String(),
		Transport: TransportTypeTCP,
	}
	var conn net.Conn
	conn, err = nac.Dial(context.Background())
	require.NoError(t, err)
	_, err = conn.Write([]byte("test"))
	assert.NoError(t, err)
	assert.NoError(t, conn.Close())
	<-done
	assert.NoError(t, ln.Close())
}

func Test_NetAddr_Validate(t *testing.T) {
	na := &AddrConfig{
		Transport: TransportTypeTCP,
	}
	require.NoError(t, na.Validate())

	na = &AddrConfig{
		Transport: transportTypeEmpty,
	}
	require.Error(t, na.Validate())

	na = &AddrConfig{
		Transport: "random string",
	}
	assert.Error(t, na.Validate())
}

func TestTCPAddrConfig(t *testing.T) {
	nas := &TCPAddrConfig{
		Endpoint: "localhost:0",
	}
	ln, err := nas.Listen(context.Background())
	require.NoError(t, err)
	done := make(chan bool, 1)

	go func() {
		conn, errGo := ln.Accept()
		assert.NoError(t, errGo)
		buf := make([]byte, 10)
		var numChr int
		numChr, errGo = conn.Read(buf)
		assert.NoError(t, errGo)
		assert.Equal(t, "test", string(buf[:numChr]))
		assert.NoError(t, conn.Close())
		done <- true
	}()

	nac := &TCPAddrConfig{
		Endpoint: ln.Addr().String(),
	}
	var conn net.Conn
	conn, err = nac.Dial(context.Background())
	require.NoError(t, err)
	_, err = conn.Write([]byte("test"))
	assert.NoError(t, err)
	assert.NoError(t, conn.Close())
	<-done
	assert.NoError(t, ln.Close())
}

func Test_TransportType_UnmarshalText(t *testing.T) {
	var tt TransportType
	err := tt.UnmarshalText([]byte("tcp"))
	require.NoError(t, err)
	err = tt.UnmarshalText([]byte("invalid"))
	require.Error(t, err)
}

func TestServerReusePort(t *testing.T) {
	if runtime.GOOS == "windows" {
		sc := &AddrConfig{
			Endpoint:  "localhost:4318",
			Transport: TransportTypeTCP,
			ReusePort: true,
		}

		_, err := sc.Listen(t.Context())
		require.Error(t, err, "ReusePort is not supported on Windows")
		return
	}

	tests := []struct {
		name          string
		reusePort     bool
		expectedError bool
	}{
		{
			name:          "ReusePort enabled",
			reusePort:     true,
			expectedError: false,
		},
		{
			name:          "ReusePort disabled",
			reusePort:     false,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &AddrConfig{
				Endpoint:  "localhost:4318",
				Transport: TransportTypeTCP,
				ReusePort: tt.reusePort,
			}

			ln1, err := sc.Listen(t.Context())
			require.NoError(t, err)
			defer ln1.Close()

			ln2, err := sc.Listen(t.Context())
			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if ln2 != nil {
				ln2.Close()
			}
		})
	}
}

func TestServerConfigValidate(t *testing.T) {
	sc := &AddrConfig{
		Endpoint:  "localhost:4318",
		Transport: TransportTypeTCP,
		ReusePort: true,
	}

	if runtime.GOOS == "windows" {
		require.Error(t, sc.Validate())
	} else {
		require.NoError(t, sc.Validate())
	}
}
