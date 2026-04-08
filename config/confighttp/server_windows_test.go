// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/confignet"
)

func TestHTTPServerTransportNpipe(t *testing.T) {
	addr := fmt.Sprintf("\\\\.\\pipe\\%s", t.Name())
	sc := &ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  addr,
			Transport: confignet.TransportTypeNpipe,
		},
	}
	ln, err := sc.ToListener(t.Context())
	require.NoError(t, err)
	startServer(t, sc, ln, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return winio.DialPipeContext(ctx, addr)
			},
		},
		Timeout: 5 * time.Second, // Set a client-level timeout
	}
	resp, err := client.Get("http://whatever/foo")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
