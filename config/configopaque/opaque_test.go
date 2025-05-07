// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configopaque // import "go.opentelemetry.io/collector/config/configopaque"

import (
	"encoding"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ encoding.TextMarshaler = String("")

var _ fmt.Stringer = String("")

var _ fmt.GoStringer = String("")

var _ encoding.BinaryMarshaler = String("")

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
	assert.JSONEq(t, `{"opaque":"[REDACTED]","plain":"plain"}`, string(bytes))
}

func TestStringFmt(t *testing.T) {
	examples := []String{"opaque", "s", "veryveryveryveryveryveryveryveryveryverylong"}
	verbs := []string{"%s", "%q", "%v", "%#v", "%+v", "%x"}
	for _, example := range examples {
		for _, verb := range verbs {
			t.Run(fmt.Sprintf("%s/%s", string(example), verb), func(t *testing.T) {
				assert.Equal(t,
					fmt.Sprintf(verb, maskedString),
					fmt.Sprintf(verb, example),
				)
			})
		}

		for _, verb := range verbs {
			t.Run(fmt.Sprintf("string(%s)/%s", string(example), verb), func(t *testing.T) {
				// converting to a string allows to output as an unredacted string still:
				var expected string
				switch verb {
				case "%s", "%v", "%+v":
					expected = string(example)
				case "%q", "%#v":
					expected = "\"" + string(example) + "\""
				case "%x":
					expected = hex.EncodeToString([]byte(example))
				default:
					t.Errorf("unexpected verb %q", verb)
				}

				assert.Equal(t,
					expected,
					fmt.Sprintf(verb, string(example)),
				)
			})
		}
	}
}

func TestStringMarshalBinary(t *testing.T) {
	examples := []String{"opaque", "s", "veryveryveryveryveryveryveryveryveryverylong"}
	for _, example := range examples {
		opaque, err := example.MarshalBinary()
		require.NoError(t, err)
		assert.Equal(t, []byte("[REDACTED]"), opaque)
	}
}
