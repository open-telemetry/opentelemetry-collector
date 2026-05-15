// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package confignet

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNpipeListenAndDial(t *testing.T) {
	endpoint := `\\.\pipe\otel-test-confignet`
	nas := &AddrConfig{
		Endpoint:  endpoint,
		Transport: TransportTypeNpipe,
	}
	ln, err := nas.Listen(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, ln.Close())
	})
	defer ln.Close()

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
		Endpoint:  endpoint,
		Transport: TransportTypeNpipe,
	}
	var conn net.Conn
	conn, err = nac.Dial(context.Background())
	require.NoError(t, err)
	_, err = conn.Write([]byte("test"))
	assert.NoError(t, err)
	assert.NoError(t, conn.Close())
	<-done
}

func TestNpipeListenSecurityDescriptor(t *testing.T) {
	tests := []struct {
		name               string
		securityDescriptor string
		wantErr            bool
	}{
		{name: "default", securityDescriptor: "", wantErr: false},
		{name: "owner local system, allow everyone read/write", securityDescriptor: "O:SYG:SYD:(A;;GRGW;;;WD)", wantErr: false},
		{name: "invalid SDDL", securityDescriptor: "not-a-valid-sddl", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nas := &AddrConfig{
				Endpoint:  `\\.\pipe\otel-test-confignet-sd`,
				Transport: TransportTypeNpipe,
				NpipeConfig: NpipeConfig{
					SecurityDescriptor: tt.securityDescriptor,
				},
			}
			ln, err := nas.Listen(t.Context())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NoError(t, ln.Close())
		})
	}
}
