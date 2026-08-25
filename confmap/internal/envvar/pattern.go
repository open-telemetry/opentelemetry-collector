// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package envvar // import "go.opentelemetry.io/collector/confmap/internal/envvar"

import "regexp"

const ValidationPattern = `^[a-zA-Z_][a-zA-Z0-9_]*$`

var ValidationRegexp = regexp.MustCompile(ValidationPattern)
