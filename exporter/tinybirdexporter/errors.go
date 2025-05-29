// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package tinybirdexporter

import "errors"

var (
	// errMissingToken indicates that the Tinybird API token is missing.
	errMissingToken = errors.New("missing Tinybird API token")

	// errMissingDataSource indicates that the Tinybird data source name is missing.
	errMissingDataSource = errors.New("missing Tinybird data source name")

	// errMissingEndpoint indicates that the Tinybird API endpoint is missing.
	errMissingEndpoint = errors.New("missing Tinybird API endpoint")
)
