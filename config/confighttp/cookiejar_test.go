// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
)

func setCookieHeaders(t *testing.T, lines ...string) []*http.Cookie {
	t.Helper()
	resp := http.Response{Header: http.Header{"Set-Cookie": lines}}
	return resp.Cookies()
}

func cookieNames(cookies []*http.Cookie) []string {
	names := make([]string, 0, len(cookies))
	for _, c := range cookies {
		names = append(names, c.Name)
	}
	return names
}

func TestToClientBoundedCookieJar(t *testing.T) {
	cc := NewDefaultClientConfig()
	cc.Endpoint = "http://localhost:1234"
	cc.Cookies = configoptional.Some(CookiesConfig{})

	client, err := cc.ToClient(t.Context(), nil, componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	assert.IsType(t, &boundedCookieJar{}, client.Jar)
}

func TestBoundedCookieJarDropsLargeCookies(t *testing.T) {
	jar, err := newBoundedCookieJar()
	require.NoError(t, err)
	u, err := url.Parse("http://example.com/v1/traces")
	require.NoError(t, err)

	jar.SetCookies(u, setCookieHeaders(t,
		"small=value",
		"large="+strings.Repeat("A", maxCookieBytes),
	))
	assert.Equal(t, []string{"small"}, cookieNames(jar.Cookies(u)))
}

func TestBoundedCookieJarLimitsCookieCount(t *testing.T) {
	jar, err := newBoundedCookieJar()
	require.NoError(t, err)
	u, err := url.Parse("http://example.com/v1/traces")
	require.NoError(t, err)

	jar.SetCookies(u, setCookieHeaders(t, "once=1"))
	// A cookie set again with every response counts once.
	for range 2 * maxCookies {
		jar.SetCookies(u, setCookieHeaders(t, "session=1"))
	}
	assert.ElementsMatch(t, []string{"once", "session"}, cookieNames(jar.Cookies(u)))

	for i := range maxCookies - 2 {
		jar.SetCookies(u, setCookieHeaders(t, fmt.Sprintf("c%d=1", i)))
	}
	assert.Len(t, jar.stored, maxCookies)
	assert.Len(t, jar.Cookies(u), maxCookies)

	// One more distinct cookie empties the jar before storing the response's cookies.
	jar.SetCookies(u, setCookieHeaders(t, "session=2", "new=1"))
	assert.Len(t, jar.stored, 2)
	assert.ElementsMatch(t, []string{"session", "new"}, cookieNames(jar.Cookies(u)))
}
