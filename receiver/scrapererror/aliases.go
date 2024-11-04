// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package scrapererror provides aliases only for go.opentelemetry.io/receiver/scraper/scrapererror
// It will be deleted in a future version.
package scrapererror // import "go.opentelemetry.io/collector/receiver/scrapererror"

import "go.opentelemetry.io/collector/receiver/scraper/scrapererror"

type (
	ScrapeErrors       = scrapererror.ScrapeErrors
	PartialScrapeError = scrapererror.PartialScrapeError
)

var (
	AddPartial            = (*ScrapeErrors).AddPartial
	Add                   = (*ScrapeErrors).Add
	Combine               = (*ScrapeErrors).Combine
	NewPartialScrapeError = scrapererror.NewPartialScrapeError
	IsPartialScrapeError  = scrapererror.IsPartialScrapeError
)
