// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"context"
	"net"
	"net/http"

	"go.opentelemetry.io/collector/consumer/consumerconnection"
)

// clientInfoHandler is an http.Handler that enhances the incoming request context with consumerconnection.Info.
type clientInfoHandler struct {
	next http.Handler

	// include client metadata or not
	includeMetadata bool
}

// ServeHTTP intercepts incoming HTTP requests, replacing the request's context with one that contains
// a consumerconnection.Info containing the client's IP address.
func (h *clientInfoHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	req = req.WithContext(contextWithClient(req, h.includeMetadata))
	h.next.ServeHTTP(w, req)
}

// contextWithClient attempts to add the client IP address to the consumerconnection.Info from the context. When no
// consumerconnection.Info exists in the context, one is created.
func contextWithClient(req *http.Request, includeMetadata bool) context.Context {
	cl := consumerconnection.InfoFromContext(req.Context())

	ip := parseIP(req.RemoteAddr)
	if ip != nil {
		cl.Addr = ip
	}

	if includeMetadata {
		md := req.Header.Clone()
		if len(md.Get(consumerconnection.MetadataHostName)) == 0 && req.Host != "" {
			md.Add(consumerconnection.MetadataHostName, req.Host)
		}

		cl.Metadata = consumerconnection.NewMetadata(md)
	}

	ctx := consumerconnection.NewContextWithInfo(req.Context(), cl)
	return ctx
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
