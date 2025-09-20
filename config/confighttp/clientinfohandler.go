// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"context"
	"net"
	"net/http"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/config/configtls"
)

// clientInfoHandler is an http.Handler that enhances the incoming request context with client.Info.
type clientInfoHandler struct {
	next http.Handler

	// include client metadata or not
	includeMetadata bool

	// include tls metadata or not
	includeTLSMetadata bool
}

// ServeHTTP intercepts incoming HTTP requests, replacing the request's context with one that contains
// a client.Info containing the client's IP address.
func (h *clientInfoHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	req = req.WithContext(contextWithClient(req, h.includeMetadata, h.includeTLSMetadata)) //nolint:contextcheck //context already handled through contextWithClient
	h.next.ServeHTTP(w, req)
}

// contextWithClient attempts to add the client IP address to the client.Info from the context. When no
// client.Info exists in the context, one is created.
func contextWithClient(req *http.Request, includeMetadata, includeTLSMetadata bool) context.Context {
	cl := client.FromContext(req.Context())

	ip := parseIP(req.RemoteAddr)
	if ip != nil {
		cl.Addr = ip
	}

	var md http.Header
	if includeMetadata {
		md = req.Header.Clone()
		if md.Get(client.MetadataHostName) == "" && req.Host != "" {
			md.Add(client.MetadataHostName, req.Host)
		}
		cl.Metadata = client.NewMetadata(md)
	}
	if includeTLSMetadata {
		if md == nil {
			md = make(http.Header)
		}
		if req.TLS != nil && len(req.TLS.PeerCertificates) > 0 {
			tlsMd := configtls.MetadataFromPeerCert(req.TLS.PeerCertificates[0])
			for key, vals := range tlsMd {
				for _, v := range vals {
					md.Add(key, v)
				}
			}
			cl.Metadata = client.NewMetadata(md)
		}
	}

	ctx := client.NewContext(req.Context(), cl)
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
