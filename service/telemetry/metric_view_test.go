// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func strToPtr(input string) *string {
	return &input
}

func TestViewOptionsFromConfig(t *testing.T) {
	for _, tc := range []struct {
		name     string
		views    []View
		expected int
	}{
		{
			name:  "empty views",
			views: []View{},
		},
		{
			name: "nil selector",
			views: []View{
				{},
			},
		},
		{
			name: "all instruments",
			views: []View{
				{
					Selector: &ViewSelector{
						InstrumentName: strToPtr("counter_instrument"),
						InstrumentType: strToPtr("counter"),
						MeterName:      strToPtr("meter-1"),
						MeterVersion:   strToPtr("0.1.0"),
						MeterSchemaUrl: strToPtr("http://schema123"),
					},
					Stream: &ViewStream{
						Name:        strToPtr("new-stream"),
						Description: strToPtr("new-description"),
					},
				},
				{
					Selector: &ViewSelector{
						InstrumentName: strToPtr("histogram_instrument"),
						InstrumentType: strToPtr("histogram"),
						MeterName:      strToPtr("meter-1"),
						MeterVersion:   strToPtr("0.1.0"),
						MeterSchemaUrl: strToPtr("http://schema123"),
					},
					Stream: &ViewStream{
						Name:        strToPtr("new-stream"),
						Description: strToPtr("new-description"),
					},
				},
				{
					Selector: &ViewSelector{
						InstrumentName: strToPtr("observable_counter_instrument"),
						InstrumentType: strToPtr("observable_counter"),
						MeterName:      strToPtr("meter-1"),
						MeterVersion:   strToPtr("0.1.0"),
						MeterSchemaUrl: strToPtr("http://schema123"),
					},
					Stream: &ViewStream{
						Name:        strToPtr("new-stream"),
						Description: strToPtr("new-description"),
					},
				},
				{
					Selector: &ViewSelector{
						InstrumentName: strToPtr("observable_gauge_instrument"),
						InstrumentType: strToPtr("observable_gauge"),
						MeterName:      strToPtr("meter-1"),
						MeterVersion:   strToPtr("0.1.0"),
						MeterSchemaUrl: strToPtr("http://schema123"),
					},
					Stream: &ViewStream{
						Name:        strToPtr("new-stream"),
						Description: strToPtr("new-description"),
					},
				},
				{
					Selector: &ViewSelector{
						InstrumentName: strToPtr("observable_updown_counter_instrument"),
						InstrumentType: strToPtr("observable_updown_counter"),
						MeterName:      strToPtr("meter-1"),
						MeterVersion:   strToPtr("0.1.0"),
						MeterSchemaUrl: strToPtr("http://schema123"),
					},
					Stream: &ViewStream{
						Name:        strToPtr("new-stream"),
						Description: strToPtr("new-description"),
					},
				},
				{
					Selector: &ViewSelector{
						InstrumentName: strToPtr("updown_counter_instrument"),
						InstrumentType: strToPtr("updown_counter"),
						MeterName:      strToPtr("meter-1"),
						MeterVersion:   strToPtr("0.1.0"),
						MeterSchemaUrl: strToPtr("http://schema123"),
					},
					Stream: &ViewStream{
						Name:        strToPtr("new-stream"),
						Description: strToPtr("new-description"),
					},
				},
			},
			expected: 6,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, len(ViewOptionsFromConfig(tc.views)), tc.expected)
		})
	}
}
