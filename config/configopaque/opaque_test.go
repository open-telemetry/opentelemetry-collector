// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configopaque // import "go.opentelemetry.io/collector/config/configopaque"

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestStringMarshalText(t *testing.T) {
	examples := []String{"opaque", "s", "veryveryveryveryveryveryveryveryveryverylong"}
	for _, example := range examples {
		opaque, err := example.MarshalText()
		require.NoError(t, err)
		assert.Equal(t, maskedString, string(opaque))
	}
}

type TestStruct struct {
	Opaque String `json:"opaque" yaml:"opaque"`
	Plain  string `json:"plain" yaml:"plain"`
}

var example = TestStruct{
	Opaque: "opaque",
	Plain:  "plain",
}

func TestStringJSON(t *testing.T) {
	bytes, err := json.Marshal(example)
	require.NoError(t, err)
	assert.Equal(t, `{"opaque":"[REDACTED]","plain":"plain"}`, string(bytes))
}

func TestStringYAML(t *testing.T) {
	bytes, err := yaml.Marshal(example)
	require.NoError(t, err)
	assert.Equal(t, "opaque: '[REDACTED]'\nplain: plain\n", string(bytes))
}

func TestStringFmt(t *testing.T) {
	examples := []String{"opaque", "s", "veryveryveryveryveryveryveryveryveryverylong"}
	verbTests := []struct {
		verb string
		want string
	}{
		{verb: "%s", want: maskedString},
		{verb: "%q", want: fmt.Sprintf("%q", maskedString)},
		{verb: "%v", want: maskedString},
		{verb: "%#v", want: maskedString},
		{verb: "%+v", want: maskedString},
		{verb: "%x", want: fmt.Sprintf("%x", maskedString)},
	}
	for _, example := range examples {
		for _, tt := range verbTests {
			t.Run(fmt.Sprintf("%s/%s", string(example), tt.verb), func(t *testing.T) {
				assert.Equal(t, tt.want, fmt.Sprintf(tt.verb, example))
			})
		}
		// converting to a string allows to output as an unredacted string still:
		// nolint S1025
		assert.Equal(t, string(example), fmt.Sprintf("%s", string(example)))
	}
}
