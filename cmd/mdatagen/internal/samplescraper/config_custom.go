// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package samplescraper // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/samplescraper"

import (
	"go.opentelemetry.io/collector/config/confighttp"
)

func validateHTTPClient(_ *confighttp.ClientConfig) error {
	return nil
}
