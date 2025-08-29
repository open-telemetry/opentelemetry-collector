// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import (
	"crypto/x509"

	"go.opentelemetry.io/collector/client"
)

func MetadataFromPeerCert(c *x509.Certificate) map[string][]string {
	md := make(map[string][]string)
	md[client.MetadataTLSPeerSubject] = []string{c.Subject.String()}
	if len(c.URIs) > 0 {
		vals := make([]string, len(c.URIs))
		for i, uri := range c.URIs {
			vals[i] = uri.String()
		}
		md[client.MetadataTLSPeerURI] = vals
	}
	return md
}
