// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"testing"

	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
)

func TestPcommonResourceFromConfigAttributesListError(t *testing.T) {
	list := "invalid"
	_, err := pcommonResourceFromConfig(config.Resource{AttributesList: &list})
	require.ErrorContains(t, err, "resource attributes_list has missing value")
}
