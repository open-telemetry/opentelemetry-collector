// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofileotlp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	_ json.Unmarshaler = ExportRequest{}
	_ json.Marshaler   = ExportRequest{}
)

var profilesRequestJSON = []byte(`
	{
		"resourceProfiles": [
			{
				"resource": {},
				"scopeProfiles": [
					{
						"scope": {},
						"profiles": [
							{
								"sample": [
									{
										"locationsStartIndex": 42
									}
								],
								"periodType": {}
							}
						]
					}
				]
			}
		],
		"dictionary": {}
	}`)

func TestRequestToPData(t *testing.T) {
	tr := NewExportRequest()
	assert.Equal(t, 0, tr.Profiles().SampleCount())
	tr.Profiles().ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty().Sample().AppendEmpty()
	assert.Equal(t, 1, tr.Profiles().SampleCount())
}

func TestRequestJSON(t *testing.T) {
	tr := NewExportRequest()
	require.NoError(t, tr.UnmarshalJSON(profilesRequestJSON))
	assert.Equal(t, int32(42), tr.Profiles().ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0).Sample().At(0).LocationsStartIndex())

	got, err := tr.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, strings.Join(strings.Fields(string(profilesRequestJSON)), ""), string(got))
}
