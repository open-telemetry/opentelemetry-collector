// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"context"
	"net"
	"net/http"

	"go.opentelemetry.io/collector/client"
)

// clientInfoHandler is an http.Handler that enhances the incoming request context with client.Info.
type clientInfoHandler struct {
	next http.Handler

	// include client metadata or not
	includeMetadata bool

	// metadata keys to determine the client address
	clientAddrMetadataKeys []string
}

// ServeHTTP intercepts incoming HTTP requests, replacing the request's context with one that contains
// a client.Info containing the client's IP address.
func (h *clientInfoHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	req = req.WithContext(contextWithClient(req, h.includeMetadata, h.clientAddrMetadataKeys)) //nolint:contextcheck //context already handled through contextWithClient
	h.next.ServeHTTP(w, req)
}

// contextWithClient attempts to add the client IP address to the client.Info from the context.
// The address is found by first checking the metadata using clientAddrMetadataKeys and
// falls back to the request Remote address
// When no client.Info exists in the context, one is created.
func contextWithClient(req *http.Request, includeMetadata bool, clientAddrMetadataKeys []string) context.Context {
	cl := client.FromContext(req.Context())

	var ip *net.IPAddr
	if ip = getIP(req.Header, clientAddrMetadataKeys); ip == nil {
		ip = parseIP(req.RemoteAddr)
	}
	if ip != nil {
		cl.Addr = ip
	}

	if includeMetadata {
		md := req.Header.Clone()
		if md.Get(client.MetadataHostName) == "" && req.Host != "" {
			md.Add(client.MetadataHostName, req.Host)
		}

		cl.Metadata = client.NewMetadata(md)
	}

	ctx := client.NewContext(req.Context(), cl)
	return ctx
}

// getIP checks keys in order to get an IP address.
// Returns the first valid IP address found, otherwise
// returns nil.
func getIP(header http.Header, keys []string) *net.IPAddr {
	for _, key := range keys {
		addr := header.Get(key)
		if addr != "" {
			if ip := parseIP(addr); ip != nil {
				return ip
			}
		}
	}
	return nil
}

// parseIP parses the given string for an IP address. The input string might contain the port,
// but must not contain a protocol or path. Suitable for getting the IP part of a client connection.
func parseIP(source string) *net.IPAddr {
	ipstr, _, err := net.SplitHostPort(source)
	if err == nil {
		source = ipstr
	}
	ip := net.ParseIP(source)
	if ip != nil {
		return &net.IPAddr{
			IP: ip,
		}
	}
	return nil
}
