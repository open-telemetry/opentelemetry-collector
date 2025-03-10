// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package admissionlimiterextension // import "go.opentelemetry.io/collector/extension/admissionlimiterextension"

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/limit"
)

type admissionLimiterExtension struct {
}

var _ limit.Extension = &admissionLimiterExtension{}

// newAdmissionlimiter returns a new admissionlimiter extension.
func newAdmissionlimiter(cfg *Config, logger *zap.Logger) (*admissionLimiterExtension, error) {
	// This will allocate an instance of the BoundedQueue logic in
	// https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/internal/otelarrow/admission2

	return &admissionLimiterExtension{}, nil
}

func (al *admissionLimiterExtension) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (al *admissionLimiterExtension) Shutdown(ctx context.Context) error {
	return nil
}

func (al *admissionLimiterExtension) GetClient(ctx context.Context, kind component.Kind, id component.ID) (limit.Client, error) {

}
