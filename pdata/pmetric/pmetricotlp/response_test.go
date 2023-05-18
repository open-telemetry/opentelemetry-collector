// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetricotlp

import (
	"encoding/json"
)

var _ json.Unmarshaler = ExportResponse{}
var _ json.Marshaler = ExportResponse{}
