// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/confmap/provider/internal"

import (
	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/confmap"
)

// NewRetrievedFromYAML returns a new Retrieved instance that contains the deserialized data from the yaml bytes.
// * yamlBytes the yaml bytes that will be deserialized.
// * opts specifies options associated with this Retrieved value, such as CloseFunc.
func NewRetrievedFromYAML(yamlBytes []byte, opts ...confmap.RetrievedOption) (*confmap.Retrieved, error) {
	var rawConf any
	if err := yaml.Unmarshal(yamlBytes, &rawConf); err != nil {
		return nil, err
	}
	return confmap.NewRetrieved(rawConf, opts...)
}
