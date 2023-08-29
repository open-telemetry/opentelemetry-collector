package scrapererror

import (
	"errors"

	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type ScrapeLevelErrors struct {
	Errors        []error
	WarningErrors []error
	DebugErrors   []error
}

func (s *ScrapeLevelErrors) Error() string {
	err := multierr.Combine(s.Errors...)
	if err != nil {
		return err.Error()
	}
	return ""
}

func (s *ScrapeLevelErrors) AddPartial(failed int, err error) {
	s.Errors = append(s.Errors, NewPartialScrapeError(err, failed))
}

func (s *ScrapeLevelErrors) Add(err error) {
	s.Errors = append(s.Errors, err)
}

func (s *ScrapeLevelErrors) AddPartialWarning(failed int, err error) {
	s.WarningErrors = append(s.WarningErrors, NewPartialScrapeError(err, failed))
}

func (s *ScrapeLevelErrors) AddWarning(err error) {
	s.WarningErrors = append(s.WarningErrors, err)
}

func (s *ScrapeLevelErrors) AddPartialDebug(failed int, err error) {
	s.DebugErrors = append(s.DebugErrors, NewPartialScrapeError(err, failed))
}

func (s *ScrapeLevelErrors) AddDebug(err error) {
	s.DebugErrors = append(s.DebugErrors, err)
}

func (s *ScrapeLevelErrors) CombineErrors() error {
	if len(s.Errors) == 0 {
		return nil
	}

	sumOfFailed := 0
	isPartialScrapeError := false
	for _, err := range s.Errors {
		var partial PartialScrapeError
		if errors.As(err, &partial) {
			isPartialScrapeError = true
			sumOfFailed += partial.Failed
		}
	}

	combined := multierr.Combine(s.Errors...)
	if !isPartialScrapeError {
		return combined
	}

	if combined == nil {
		return nil
	}

	return NewPartialScrapeError(combined, sumOfFailed)
}

func (s *ScrapeLevelErrors) ZapLogAll(logger *zap.Logger, scraperID string) {
	scraperIDField := zap.String("scraper", scraperID)

	errors := multierr.Combine(s.Errors...)
	if errors != nil {
		logger.Error("Error scraping metrics", zap.Error(errors), scraperIDField)
	}

	warningErrors := multierr.Combine(s.WarningErrors...)
	if warningErrors != nil {
		logger.Warn("Warning scraping metrics", zap.NamedError("warning", warningErrors), scraperIDField)
	}

	debugErrors := multierr.Combine(s.DebugErrors...)
	if debugErrors != nil {
		logger.Debug("Debug information scraping metrics", zap.NamedError("debug", debugErrors), scraperIDField)
	}
}
