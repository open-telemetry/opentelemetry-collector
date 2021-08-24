// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filterprocessor

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/internal/processor/filterconfig"
	"go.opentelemetry.io/collector/internal/processor/filtermatcher"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

type filterLogProcessor struct {
	cfg              *Config
	excludeResources filtermatcher.AttributesMatcher
	logger           *zap.Logger
}

func newFilterLogsProcessor(logger *zap.Logger, cfg *Config) (*filterLogProcessor, error) {
	excludeResources, err := createLogsMatcher(cfg.Logs.ResourceAttributes)
	if err != nil {
		logger.Error(
			"filterlog: Error creating logs resources matcher",
			zap.Error(err),
		)
		return nil, err
	}

	return &filterLogProcessor{
		cfg:              cfg,
		excludeResources: excludeResources,
		logger:           logger,
	}, nil
}

func createLogsMatcher(excludeResources []filterconfig.Attribute) (filtermatcher.AttributesMatcher, error) {
	// Nothing specified in configuration
	if excludeResources == nil {
		return nil, nil
	}
	var attributeMatcher filtermatcher.AttributesMatcher
	attributeMatcher, err := filtermatcher.NewAttributesMatcher(
		filterset.Config{
			MatchType: filterset.MatchType("strict"),
		},
		excludeResources,
	)
	if err != nil {
		return attributeMatcher, err
	}
	return attributeMatcher, nil
}

func (flp *filterLogProcessor) ProcessLogs(ctx context.Context, logs pdata.Logs) (pdata.Logs, error) {
	logs.ResourceLogs().RemoveIf(func(rm pdata.ResourceLogs) bool {
		return flp.shouldSkipLogsForResource(rm.Resource())
	})

	if logs.ResourceLogs().Len() == 0 {
		return logs, processorhelper.ErrSkipProcessingData
	}

	return logs, nil
}

// shouldSkipLogsForResource determines if a log should be processed.
// True is returned when a log should be skipped.
// False is returned when a log should not be skipped.
// The logic determining if a log should be skipped is set in the resource attribute configuration.
func (flp *filterLogProcessor) shouldSkipLogsForResource(resource pdata.Resource) bool {
	resourceAttributes := resource.Attributes()

	if flp.excludeResources != nil {
		matches := flp.excludeResources.Match(resourceAttributes)
		if matches {
			return true
		}
	}

	return false
}
