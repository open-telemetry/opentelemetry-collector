// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log/logtest"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/internal/telemetry/componentattribute"
	"go.opentelemetry.io/collector/pipeline"
)

type scopedRecord struct {
	s *logtest.ScopeRecords
	r logtest.EmittedRecord
}

func TestCore(t *testing.T) {
	lp := logtest.NewRecorder()
	logger := zap.New(componentattribute.NewServiceZapCore(lp, "testinstr", nil, attribute.NewSet()))

	attrs := attribute.NewSet(
		attribute.String(componentattribute.SignalKey, pipeline.SignalLogs.String()),
		attribute.String(componentattribute.ComponentIDKey, "filelog"),
	)

	parent := componentattribute.ZapLoggerWithAttributes(logger, attrs)
	parent.Info("1. test parent before child")
	child := componentattribute.ZapLoggerWithAttributes(parent, componentattribute.RemoveAttributes(attrs, componentattribute.SignalKey))
	child.Info("2. test child")
	parent.Info("3. test parent after child")

	observedScopes := lp.Result()
	var observedLogs []scopedRecord
	for _, scope := range observedScopes {
		require.Equal(t, "testinstr", scope.Name)
		for _, record := range scope.Records {
			observedLogs = append(observedLogs, scopedRecord{s: scope, r: record})
		}
	}
	slices.SortFunc(observedLogs, func(r1 scopedRecord, r2 scopedRecord) int {
		return strings.Compare(r1.r.Body().String(), r2.r.Body().String())
	})
	require.Len(t, observedLogs, 3)

	childAttrs := attribute.NewSet(
		attribute.String(componentattribute.ComponentIDKey, "filelog"),
	)

	assert.Equal(t, "1. test parent before child", observedLogs[0].r.Body().String())
	assert.Equal(t, attrs, observedLogs[0].s.Attributes)

	assert.Equal(t, "2. test child", observedLogs[1].r.Body().String())
	assert.Equal(t, childAttrs, observedLogs[1].s.Attributes)

	assert.Equal(t, "3. test parent after child", observedLogs[2].r.Body().String())
	assert.Equal(t, attrs, observedLogs[2].s.Attributes)
}
