// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package samplescraper // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/samplescraper"
import "go.opentelemetry.io/collector/component"

func validateJobName(_ string) error {
	return nil
}

func validateComponentID(_ component.ID) error {
	return nil
}
