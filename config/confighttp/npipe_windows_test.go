// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package confighttp

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/confignet"
)

func testHttpServerNpipeTransport(t *testing.T) {
	t.Run("npipe", func(t *testing.T) {
		addr := `\\.\pipe\otel-test-confighttp-` + t.Name()
		sc := &ServerConfig{
			NetAddr: confignet.AddrConfig{
				Endpoint:  addr,
				Transport: confignet.TransportTypeNpipe,
			},
		}
		ln, err := sc.ToListener(context.Background())
		require.NoError(t, err)
		startServer(t, sc, ln, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, "npipe-ok")
		}))

		client := http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return winio.DialPipeContext(ctx, addr)
				},
			},
			Timeout: 5 * time.Second,
		}
		resp, err := client.Get("http://npipe/")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "npipe-ok", string(body))
	})
}
