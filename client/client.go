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

// Package client contains generic representations of clients connecting to different receivers
package client

import (
	"context"
	"net"
	"net/http"
	"regexp"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

type ctxKey struct{}

const AuthTokenHeader = "Authorization"

var TokenRegex = *regexp.MustCompile(`/([^/]{120,})$`)

// Client represents a generic client that sends data to any receiver supported by the OT receiver
type Client struct {
	IP    string
	Token string
}

// NewContext takes an existing context and derives a new context with the client value stored on it
func NewContext(ctx context.Context, c *Client) context.Context {
	return context.WithValue(ctx, ctxKey{}, c)
}

// FromContext takes a context and returns a Client value from it, if present.
func FromContext(ctx context.Context) (*Client, bool) {
	c, ok := ctx.Value(ctxKey{}).(*Client)
	return c, ok
}

// FromGRPC takes a GRPC context and tries to extract client information from it
func FromGRPC(ctx context.Context) (*Client, bool) {
	var ip, token string

	if p, ok := peer.FromContext(ctx); ok {
		ip = parseIP(p.Addr.String())
	}
	if m, ok := metadata.FromIncomingContext(ctx); ok {
		vals := m.Get(AuthTokenHeader)
		if vals != nil {
			token = vals[0]
		}
	}
	return &Client{ip, token}, true
}

// FromHTTP takes a net/http Request object and tries to extract client information from it
func FromHTTP(r *http.Request) (*Client, bool) {
	ip := parseIP(r.RemoteAddr)
	token := r.Header.Get(AuthTokenHeader)

	if token == "" {
		token = tokenFromURI(r.RequestURI)
	}
	return &Client{ip, token}, true
}

func tokenFromURI(uri string) string {
	res := TokenRegex.FindAllStringSubmatch(uri, -1)
	if len(res) == 1 && len(res[0]) == 2 {
		return res[0][1]
	}
	return ""
}

func parseIP(source string) string {
	ipstr, _, err := net.SplitHostPort(source)
	if err == nil {
		return ipstr
	}
	ip := net.ParseIP(source)
	if ip != nil {
		return ip.String()
	}
	return ""
}
