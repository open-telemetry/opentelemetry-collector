// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package scrapererror provides aliases for package scraper/scrapererror.
// Deprecated: [v0.115.0] Use `scraper/scrapererror` instead.
package scrapererror // import "go.opentelemetry.io/collector/receiver/scrapererror"

import "go.opentelemetry.io/collector/scraper/scrapererror"

type (
	ScrapeErrors       = scrapererror.ScrapeErrors
	PartialScrapeError = scrapererror.PartialScrapeError
)

var (
	AddPartial = (*ScrapeErrors).AddPartial
	Add        = (*ScrapeErrors).Add
	Combine    = (*ScrapeErrors).Combine
)

var (
	NewPartialScrapeError = scrapererror.NewPartialScrapeError
	IsPartialScrapeError  = scrapererror.IsPartialScrapeError
)
