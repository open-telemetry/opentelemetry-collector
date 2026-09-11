// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"context"
	"net"
	"net/http"
	"net/netip"

	"go.opentelemetry.io/collector/client"
)

// clientInfoHandler is an http.Handler that enhances the incoming request context with client.Info.
type clientInfoHandler struct {
	next http.Handler

	// include client metadata or not
	includeMetadata bool
}

// ServeHTTP intercepts incoming HTTP requests, replacing the request's context with one that contains
// a client.Info containing the client's IP address.
func (h *clientInfoHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	req = req.WithContext(contextWithClient(req, h.includeMetadata)) //nolint:contextcheck //context already handled through contextWithClient
	h.next.ServeHTTP(w, req)
}

// contextWithClient attempts to add the client IP address to the client.Info from the context. When no
// client.Info exists in the context, one is created.
func contextWithClient(req *http.Request, includeMetadata bool) context.Context {
	cl := client.FromContext(req.Context())

	ip := parseIP(req.RemoteAddr)
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

// parseIP parses the given string for an IP address. The input string might contain the port,
// but must not contain a protocol or path. Suitable for getting the IP part of a client connection.
// It supports IPv6 addresses with a zone identifier (e.g. "fe80::1%eth0"), which net.ParseIP does
// not.
func parseIP(source string) *net.IPAddr {
	if addrPort, err := netip.ParseAddrPort(source); err == nil {
		return &net.IPAddr{IP: addrPort.Addr().AsSlice(), Zone: addrPort.Addr().Zone()}
	}
	if addr, err := netip.ParseAddr(source); err == nil {
		return &net.IPAddr{IP: addr.AsSlice(), Zone: addr.Zone()}
	}
	return nil
}
