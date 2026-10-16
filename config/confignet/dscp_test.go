// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confignet

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDialerConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		dscp    int
		wantErr bool
	}{
		{name: "zero_value", dscp: 0, wantErr: false},
		{name: "min_valid", dscp: 1, wantErr: false},
		{name: "max_valid", dscp: 63, wantErr: false},
		{name: "ef_class", dscp: 46, wantErr: false},
		{name: "af41_class", dscp: 34, wantErr: false},
		{name: "negative", dscp: -1, wantErr: true},
		{name: "too_large", dscp: 64, wantErr: true},
		{name: "way_too_large", dscp: 255, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := &DialerConfig{DSCP: tt.dscp}
			err := dc.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid DSCP value")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDialerConfigDialer_NoDSCP(t *testing.T) {
	dc := &DialerConfig{}
	d := dc.Dialer()
	assert.Nil(t, d.Control, "Control should be nil when DSCP is 0")
}

func TestDialerConfigDialer_WithDSCP(t *testing.T) {
	dc := &DialerConfig{DSCP: 46}
	d := dc.Dialer()
	assert.NotNil(t, d.Control, "Control should be set when DSCP > 0")
}

func TestAddrConfigValidate_DSCP(t *testing.T) {
	na := &AddrConfig{
		Transport: TransportTypeTCP,
		DialerConfig: DialerConfig{
			DSCP: 64,
		},
	}
	err := na.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid DSCP value")

	na.DialerConfig.DSCP = 46
	err = na.Validate()
	assert.NoError(t, err)
}

func TestDSCPDialControl(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	done := make(chan struct{})
	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
		close(done)
	}()

	nac := &AddrConfig{
		Endpoint:  ln.Addr().String(),
		Transport: TransportTypeTCP,
		DialerConfig: DialerConfig{
			DSCP: 46,
		},
	}
	conn, err := nac.Dial(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, conn)
	conn.Close()
	<-done
}

func TestTCPAddrConfig_DSCP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	done := make(chan struct{})
	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
		close(done)
	}()

	nac := &TCPAddrConfig{
		Endpoint: ln.Addr().String(),
		DialerConfig: DialerConfig{
			DSCP: 34,
		},
	}
	conn, err := nac.Dial(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, conn)
	conn.Close()
	<-done
}

func TestDSCPDialControlFunction(t *testing.T) {
	ctrl := DSCPDialControl(46)
	assert.NotNil(t, ctrl)

	ctrl = DSCPDialControl(0)
	assert.NotNil(t, ctrl)
}
