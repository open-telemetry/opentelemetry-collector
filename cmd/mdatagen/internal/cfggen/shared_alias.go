// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfggen"

import "go.opentelemetry.io/collector/internal/schemagen"

type (
	ConfigMetadata        = schemagen.ConfigMetadata
	GoStructConfig        = schemagen.GoStructConfig
	CustomValidatorConfig = schemagen.CustomValidatorConfig
	Loader                = schemagen.Loader
	Ref                   = schemagen.Ref
	RefKind               = schemagen.RefKind
	Resolver              = schemagen.Resolver
)

const (
	External = schemagen.External
	Internal = schemagen.Internal
	Local    = schemagen.Local
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
