// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/internal/metadata"
)

func TestProfilesDictionaryRoundTrip(t *testing.T) {
	request := &internal.ExportProfilesServiceRequest{
		Dictionary: internal.ProfilesDictionary{
			StringTable: []string{"", "checkout"},
		},
		ResourceProfiles: []*internal.ResourceProfiles{
			{
				Resource: internal.Resource{
					Attributes: []internal.KeyValue{
						{
							Key: "service.name",
							Value: internal.AnyValue{
								Value: &internal.AnyValue_StringValue{StringValue: "checkout"},
							},
						},
						{
							Key: "nested",
							Value: internal.AnyValue{
								Value: &internal.AnyValue_KvlistValue{
									KvlistValue: &internal.KeyValueList{
										Values: []internal.KeyValue{
											{
												Key: "child",
												Value: internal.AnyValue{
													Value: &internal.AnyValue_StringValue{StringValue: "child-value"},
												},
											},
										},
									},
								},
							},
						},
						{
							Key: "array",
							Value: internal.AnyValue{
								Value: &internal.AnyValue_ArrayValue{
									ArrayValue: &internal.ArrayValue{
										Values: []internal.AnyValue{
											{Value: &internal.AnyValue_StringValue{StringValue: "item"}},
											{Value: &internal.AnyValue_BoolValue{BoolValue: true}},
										},
									},
								},
							},
						},
						{
							Key: "empty-value",
							Value: internal.AnyValue{
								Value: &internal.AnyValue_StringValue{StringValue: ""},
							},
						},
					},
				},
				ScopeProfiles: []*internal.ScopeProfiles{
					{
						Scope: internal.InstrumentationScope{
							Attributes: []internal.KeyValue{
								{
									Key: "scope.attr",
									Value: internal.AnyValue{
										Value: &internal.AnyValue_StringValue{StringValue: "checkout"},
									},
								},
							},
						},
					},
				},
			},
			{
				Resource: internal.Resource{
					Attributes: []internal.KeyValue{
						{
							Key: "service.name",
							Value: internal.AnyValue{
								Value: &internal.AnyValue_StringValue{StringValue: "checkout"},
							},
						},
						{
							Key: "region",
							Value: internal.AnyValue{
								Value: &internal.AnyValue_StringValue{StringValue: "eu-west"},
							},
						},
					},
				},
			},
		},
	}
	expected := internal.CopyExportProfilesServiceRequest(nil, request)

	require.NoError(t, ConvertProfilesToReferences(request))

	require.Greater(t, len(request.Dictionary.StringTable), 2)
	resourceAttrs := request.ResourceProfiles[0].Resource.Attributes
	assert.Empty(t, resourceAttrs[0].Key)
	assert.NotZero(t, resourceAttrs[0].KeyStrindex)
	valueRef, ok := resourceAttrs[0].Value.Value.(*internal.AnyValue_StringValueStrindex)
	require.True(t, ok)
	assert.Equal(t, int32(1), valueRef.StringValueStrindex)

	scopeAttr := request.ResourceProfiles[0].ScopeProfiles[0].Scope.Attributes[0]
	assert.Empty(t, scopeAttr.Key)
	_, ok = scopeAttr.Value.Value.(*internal.AnyValue_StringValueStrindex)
	assert.True(t, ok)

	secondResourceAttrs := request.ResourceProfiles[1].Resource.Attributes
	require.Len(t, secondResourceAttrs, 2)
	assert.Equal(t, resourceAttrs[0].KeyStrindex, secondResourceAttrs[0].KeyStrindex)
	secondValueRef, ok := secondResourceAttrs[0].Value.Value.(*internal.AnyValue_StringValueStrindex)
	require.True(t, ok)
	assert.Equal(t, valueRef.StringValueStrindex, secondValueRef.StringValueStrindex)
	assert.Equal(t, "service.name", request.Dictionary.StringTable[secondResourceAttrs[0].KeyStrindex])
	assert.Equal(t, "checkout", request.Dictionary.StringTable[secondValueRef.StringValueStrindex])

	serviceNameCount := 0
	checkoutCount := 0
	for _, value := range request.Dictionary.StringTable {
		switch value {
		case "service.name":
			serviceNameCount++
		case "checkout":
			checkoutCount++
		}
	}
	assert.Equal(t, 1, serviceNameCount)
	assert.Equal(t, 1, checkoutCount)

	ResolveProfilesReferences(request)

	assert.Equal(t, expected.ResourceProfiles, request.ResourceProfiles)
}

func TestConvertProfilesToReferencesInitializesStringTable(t *testing.T) {
	request := &internal.ExportProfilesServiceRequest{
		ResourceProfiles: []*internal.ResourceProfiles{
			{
				Resource: internal.Resource{
					Attributes: []internal.KeyValue{
						{
							Key: "key",
							Value: internal.AnyValue{
								Value: &internal.AnyValue_StringValue{StringValue: "value"},
							},
						},
					},
				},
			},
		},
	}

	require.NoError(t, ConvertProfilesToReferences(request))

	assert.Equal(t, []string{"", "key", "value"}, request.Dictionary.StringTable)

	ResolveProfilesReferences(request)
	assert.Equal(t, "key", request.ResourceProfiles[0].Resource.Attributes[0].Key)
	value, ok := request.ResourceProfiles[0].Resource.Attributes[0].Value.Value.(*internal.AnyValue_StringValue)
	require.True(t, ok)
	assert.Equal(t, "value", value.StringValue)
}

func TestConvertProfilesToReferencesRejectsInvalidStringTableSentinel(t *testing.T) {
	request := &internal.ExportProfilesServiceRequest{
		Dictionary: internal.ProfilesDictionary{
			StringTable: []string{"not-empty"},
		},
	}

	err := ConvertProfilesToReferences(request)

	require.EqualError(t, err, "profiles dictionary string_table[0] must be empty")
	assert.Equal(t, []string{"not-empty"}, request.Dictionary.StringTable)
}

func TestProfilesDictionaryReferenceEdges(t *testing.T) {
	calls := 0
	getStringIndex := func(string) (int32, error) {
		calls++
		return 1, nil
	}

	alreadyReference := internal.AnyValue{
		Value: &internal.AnyValue_StringValueStrindex{StringValueStrindex: 1},
	}
	require.NoError(t, ConvertProfilesAnyValueToReference(getStringIndex, &alreadyReference))
	assert.Zero(t, calls)

	emptyString := internal.AnyValue{
		Value: &internal.AnyValue_StringValue{StringValue: ""},
	}
	require.NoError(t, ConvertProfilesAnyValueToReference(getStringIndex, &emptyString))
	emptyRef, ok := emptyString.Value.(*internal.AnyValue_StringValueStrindex)
	require.True(t, ok)
	assert.Equal(t, int32(1), emptyRef.StringValueStrindex)
	assert.Equal(t, 1, calls)

	nilKVList := internal.AnyValue{Value: &internal.AnyValue_KvlistValue{}}
	require.NoError(t, ConvertProfilesAnyValueToReference(getStringIndex, &nilKVList))

	nilArray := internal.AnyValue{Value: &internal.AnyValue_ArrayValue{}}
	require.NoError(t, ConvertProfilesAnyValueToReference(getStringIndex, &nilArray))

	boolValue := internal.AnyValue{Value: &internal.AnyValue_BoolValue{BoolValue: true}}
	require.NoError(t, ConvertProfilesAnyValueToReference(getStringIndex, &boolValue))
	_, ok = boolValue.Value.(*internal.AnyValue_BoolValue)
	assert.True(t, ok)

	keyValues := []internal.KeyValue{
		{
			Value: internal.AnyValue{
				Value: &internal.AnyValue_StringValue{StringValue: "value"},
			},
		},
	}
	require.NoError(t, ConvertProfilesKeyValuesToReferences(getStringIndex, keyValues))
	assert.Zero(t, keyValues[0].KeyStrindex)
	assert.Equal(t, 2, calls)

	stringTable := []string{"", "resolved-key", "resolved-value"}
	references := []internal.KeyValue{
		{
			Key:         "inline-key",
			KeyStrindex: 1,
			Value: internal.AnyValue{
				Value: &internal.AnyValue_StringValueStrindex{StringValueStrindex: 2},
			},
		},
		{
			KeyStrindex: 99,
			Value: internal.AnyValue{
				Value: &internal.AnyValue_StringValueStrindex{StringValueStrindex: 99},
			},
		},
		{
			KeyStrindex: 1,
			Value: internal.AnyValue{
				Value: &internal.AnyValue_StringValueStrindex{},
			},
		},
	}
	ResolveProfilesKeyValueReferences(stringTable, references)

	assert.Equal(t, "inline-key", references[0].Key)
	assert.Equal(t, int32(1), references[0].KeyStrindex)
	resolved, ok := references[0].Value.Value.(*internal.AnyValue_StringValue)
	require.True(t, ok)
	assert.Equal(t, "resolved-value", resolved.StringValue)

	assert.Empty(t, references[1].Key)
	assert.Equal(t, int32(99), references[1].KeyStrindex)
	_, ok = references[1].Value.Value.(*internal.AnyValue_StringValueStrindex)
	assert.True(t, ok)

	assert.Equal(t, "resolved-key", references[2].Key)
	assert.Zero(t, references[2].KeyStrindex)
	_, ok = references[2].Value.Value.(*internal.AnyValue_StringValueStrindex)
	assert.True(t, ok)

	ResolveProfilesAnyValueReference(stringTable, &nilKVList)
	ResolveProfilesAnyValueReference(stringTable, &nilArray)
	ResolveProfilesAnyValueReference(stringTable, &boolValue)
}

func TestConvertProfilesKeyValuesToReferencesRejectsConflictingKeyRepresentations(t *testing.T) {
	keyValues := []internal.KeyValue{{
		Key:         "service.name",
		KeyStrindex: 1,
	}}
	getStringIndex := func(string) (int32, error) {
		return 2, nil
	}

	err := ConvertProfilesKeyValuesToReferences(getStringIndex, keyValues)

	require.EqualError(t, err, "attribute 0 has both key and key_strindex set")
	assert.Equal(t, "service.name", keyValues[0].Key)
	assert.Equal(t, int32(1), keyValues[0].KeyStrindex)
}

func TestProfilesDictionaryReferencesWithPooling(t *testing.T) {
	previous := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), true))
	defer func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), previous))
	}()

	getStringIndex := func(s string) (int32, error) {
		if s == "pooled-value" {
			return 1, nil
		}
		return 0, nil
	}

	value := internal.AnyValue{
		Value: &internal.AnyValue_StringValue{StringValue: "pooled-value"},
	}
	require.NoError(t, ConvertProfilesAnyValueToReference(getStringIndex, &value))

	ref, ok := value.Value.(*internal.AnyValue_StringValueStrindex)
	require.True(t, ok)
	assert.Equal(t, int32(1), ref.StringValueStrindex)

	ResolveProfilesAnyValueReference([]string{"", "pooled-value"}, &value)
	resolved, ok := value.Value.(*internal.AnyValue_StringValue)
	require.True(t, ok)
	assert.Equal(t, "pooled-value", resolved.StringValue)
}
