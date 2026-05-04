// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfggen"

import "go.opentelemetry.io/collector/internal/schemagen"

type ConfigMetadata = schemagen.ConfigMetadata
type GoStructConfig = schemagen.GoStructConfig
type CustomValidatorConfig = schemagen.CustomValidatorConfig
type Loader = schemagen.Loader
type Ref = schemagen.Ref
type RefKind = schemagen.RefKind
type Resolver = schemagen.Resolver

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
