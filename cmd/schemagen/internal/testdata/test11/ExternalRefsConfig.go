// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test11

import (
	"go.opentelemetry.io/collector/scraper/scraperhelper"

	"go.opentelemetry.io/collector/cmd/schemagen/internal/testdata/external"
)

type ExternalRefsConfig struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	external.TestConfig            `mapstructure:",squash"`
	Test                           external.TestConfig `mapstructure:"test_nested"`
}
