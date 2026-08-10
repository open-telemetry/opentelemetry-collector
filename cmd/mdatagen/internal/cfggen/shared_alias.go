// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfggen"

import "go.opentelemetry.io/collector/internal/schemagen"

type (
	ConfigMetadata        = schemagen.ConfigMetadata
	ConfigsMetadata       = schemagen.ConfigsMetadata
	GoStructConfig        = schemagen.GoStructConfig
	CustomValidatorConfig = schemagen.CustomValidatorConfig
	Loader                = schemagen.Loader
	Ref                   = schemagen.Ref
	RefKind               = schemagen.RefKind
	Resolver              = schemagen.Resolver
	SchemaType            = schemagen.SchemaType
)

const (
	External = schemagen.External
	Internal = schemagen.Internal
	Local    = schemagen.Local

	StringType  = schemagen.StringType
	BoolType    = schemagen.BoolType
	IntType     = schemagen.IntType
	Int8Type    = schemagen.Int8Type
	Int16Type   = schemagen.Int16Type
	Int32Type   = schemagen.Int32Type
	Int64Type   = schemagen.Int64Type
	UintType    = schemagen.UintType
	Uint8Type   = schemagen.Uint8Type
	Uint16Type  = schemagen.Uint16Type
	Uint32Type  = schemagen.Uint32Type
	Uint64Type  = schemagen.Uint64Type
	ByteType    = schemagen.ByteType
	RuneType    = schemagen.RuneType
	Float32Type = schemagen.Float32Type
	Float64Type = schemagen.Float64Type
	AnyType     = schemagen.AnyType
	ObjectType  = schemagen.ObjectType
	SliceType   = schemagen.SliceType
	MapType     = schemagen.MapType

	FloatType        = schemagen.FloatType
	DoubleType       = schemagen.DoubleType
	DurationType     = schemagen.DurationType
	TimeType         = schemagen.TimeType
	OpaqueStringType = schemagen.OpaqueStringType
	ComponentIDType  = schemagen.ComponentIDType
	OpaqueMapType    = schemagen.OpaqueMapType
)

var (
	ErrNotFound     = schemagen.ErrNotFound
	NewLoader       = schemagen.NewLoader
	NewRef          = schemagen.NewRef
	WithOrigin      = schemagen.WithOrigin
	LocalizeRef     = schemagen.LocalizeRef
	NewResolver     = schemagen.NewResolver
	WriteJSONSchema = schemagen.WriteJSONSchema
)
