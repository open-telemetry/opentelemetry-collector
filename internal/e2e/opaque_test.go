// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/confmap"
)

type TestStruct struct {
	Opaque configopaque.String `json:"opaque" yaml:"opaque"`
	Plain  string              `json:"plain" yaml:"plain"`
}

var example = TestStruct{
	Opaque: "opaque",
	Plain:  "plain",
}

func TestConfMapMarshalConfigOpaque(t *testing.T) {
	conf := confmap.New()
	require.NoError(t, conf.Marshal(example))
	assert.Equal(t, "[REDACTED]", conf.Get("opaque"))
	assert.Equal(t, "plain", conf.Get("plain"))
}
