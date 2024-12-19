// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconsumererror // import "go.opentelemetry.io/collector/consumer/consumererror/xconsumererror"

import (
	"go.opentelemetry.io/collector/consumer/consumererror/internal"
	"go.opentelemetry.io/collector/pdata/pentity"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

// Profiles is an error that may carry associated Profile data for a subset of received data
// that failed to be processed or sent.
type Profiles struct {
	internal.Retryable[pprofile.Profiles]
}

// NewProfiles creates a Profiles that can encapsulate received data that failed to be processed or sent.
func NewProfiles(err error, data pprofile.Profiles) error {
	return Profiles{
		Retryable: internal.Retryable[pprofile.Profiles]{
			Err:   err,
			Value: data,
		},
	}
}

// Entities is an error that may carry associated Log data for a subset of received data
// that failed to be processed or sent.
type Entities struct {
	internal.Retryable[pentity.Entities]
}

// NewEntities creates a Entities that can encapsulate received data that failed to be processed or sent.
func NewEntities(err error, data pentity.Entities) error {
	return Entities{
		Retryable: internal.Retryable[pentity.Entities]{
			Err:   err,
			Value: data,
		},
	}
}
