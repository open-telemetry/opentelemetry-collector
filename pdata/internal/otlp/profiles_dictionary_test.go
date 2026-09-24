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
		},
	}
	expected := internal.CopyExportProfilesServiceRequest(nil, request)

	ConvertProfilesToReferences(request)

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

	ConvertProfilesToReferences(request)

	assert.Equal(t, []string{"", "key", "value"}, request.Dictionary.StringTable)

	ResolveProfilesReferences(request)
	assert.Equal(t, "key", request.ResourceProfiles[0].Resource.Attributes[0].Key)
	value, ok := request.ResourceProfiles[0].Resource.Attributes[0].Value.Value.(*internal.AnyValue_StringValue)
	require.True(t, ok)
	assert.Equal(t, "value", value.StringValue)
}

func TestProfilesDictionaryReferenceEdges(t *testing.T) {
	calls := 0
	getStringIndex := func(string) int32 {
		calls++
		return 1
	}

	alreadyReference := internal.AnyValue{
		Value: &internal.AnyValue_StringValueStrindex{StringValueStrindex: 1},
	}
	ConvertProfilesAnyValueToReference(getStringIndex, &alreadyReference)
	assert.Zero(t, calls)

	emptyString := internal.AnyValue{
		Value: &internal.AnyValue_StringValue{StringValue: ""},
	}
	ConvertProfilesAnyValueToReference(getStringIndex, &emptyString)
	_, ok := emptyString.Value.(*internal.AnyValue_StringValue)
	assert.True(t, ok)
	assert.Zero(t, calls)

	nilKVList := internal.AnyValue{Value: &internal.AnyValue_KvlistValue{}}
	ConvertProfilesAnyValueToReference(getStringIndex, &nilKVList)

	nilArray := internal.AnyValue{Value: &internal.AnyValue_ArrayValue{}}
	ConvertProfilesAnyValueToReference(getStringIndex, &nilArray)

	boolValue := internal.AnyValue{Value: &internal.AnyValue_BoolValue{BoolValue: true}}
	ConvertProfilesAnyValueToReference(getStringIndex, &boolValue)
	_, ok = boolValue.Value.(*internal.AnyValue_BoolValue)
	assert.True(t, ok)

	keyValues := []internal.KeyValue{
		{
			Value: internal.AnyValue{
				Value: &internal.AnyValue_StringValue{StringValue: "value"},
			},
		},
	}
	ConvertProfilesKeyValuesToReferences(getStringIndex, keyValues)
	assert.Zero(t, keyValues[0].KeyStrindex)
	assert.Equal(t, 1, calls)

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

func TestProfilesDictionaryReferencesWithPooling(t *testing.T) {
	previous := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), true))
	defer func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), previous))
	}()

	getStringIndex := func(s string) int32 {
		if s == "pooled-value" {
			return 1
		}
		return 0
	}

	value := internal.AnyValue{
		Value: &internal.AnyValue_StringValue{StringValue: "pooled-value"},
	}
	ConvertProfilesAnyValueToReference(getStringIndex, &value)

	ref, ok := value.Value.(*internal.AnyValue_StringValueStrindex)
	require.True(t, ok)
	assert.Equal(t, int32(1), ref.StringValueStrindex)

	ResolveProfilesAnyValueReference([]string{"", "pooled-value"}, &value)
	resolved, ok := value.Value.(*internal.AnyValue_StringValue)
	require.True(t, ok)
	assert.Equal(t, "pooled-value", resolved.StringValue)
}
