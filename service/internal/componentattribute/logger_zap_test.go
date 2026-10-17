// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/internal/componentattribute"
)

func TestLoggerWithLevel(t *testing.T) {
	tests := []struct {
		name            string
		baseLevel       zapcore.Level
		overrideLevel   zapcore.Level
		wantDebugLogged bool
		wantWarnLogged  bool
	}{
		{
			name:            "override_debug_to_warn",
			baseLevel:       zapcore.DebugLevel,
			overrideLevel:   zapcore.WarnLevel,
			wantDebugLogged: false, // Debug is below Warn -> filtered out
			wantWarnLogged:  true,
		},
		{
			name:            "override_warn_to_debug",
			baseLevel:       zapcore.WarnLevel,
			overrideLevel:   zapcore.DebugLevel,
			wantDebugLogged: true, // Debug is >= Debug -> allowed
			wantWarnLogged:  true,
		},
		{
			name:            "override_info_to_error",
			baseLevel:       zapcore.InfoLevel,
			overrideLevel:   zapcore.ErrorLevel,
			wantDebugLogged: false,
			wantWarnLogged:  false, // Warn < Error -> filtered
		},
		{
			name:            "override_debug_to_info",
			baseLevel:       zapcore.DebugLevel,
			overrideLevel:   zapcore.InfoLevel,
			wantDebugLogged: false,
			wantWarnLogged:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, obs := observer.New(tt.baseLevel)
			baseLogger := zap.New(core)
			overriddenLogger := componentattribute.LoggerWithLevel(baseLogger, tt.overrideLevel)

			overriddenLogger.Debug("debug msg")
			overriddenLogger.Warn("warn msg")

			logs := obs.TakeAll()
			var debugSeen, warnSeen bool
			for _, l := range logs {
				switch l.Level {
				case zapcore.DebugLevel:
					debugSeen = true
				case zapcore.WarnLevel:
					warnSeen = true
				}
			}
			assert.Equal(t, tt.wantDebugLogged, debugSeen, "debug log visibility mismatch")
			assert.Equal(t, tt.wantWarnLogged, warnSeen, "warn log visibility mismatch")
		})
	}
}

func TestLoggerWithLevel_DoesNotAffectOriginal(t *testing.T) {
	// Verify LoggerWithLevel returns a NEW logger and does not mutate the original.
	core, obs := observer.New(zapcore.DebugLevel)
	baseLogger := zap.New(core)
	overriddenLogger := componentattribute.LoggerWithLevel(baseLogger, zapcore.ErrorLevel)

	// Original logger should still log debug messages (base level is Debug)
	baseLogger.Debug("original debug")
	baseLogger.Warn("original warn")

	// Overridden logger should NOT log debug/warn, only error+
	overriddenLogger.Debug("overridden debug")
	overriddenLogger.Warn("overridden warn")
	overriddenLogger.Error("overridden error")

	logs := obs.TakeAll()
	var debugCount, warnCount, errorCount int
	for _, l := range logs {
		switch l.Level {
		case zapcore.DebugLevel:
			debugCount++
		case zapcore.WarnLevel:
			warnCount++
		case zapcore.ErrorLevel:
			errorCount++
		}
	}

	// Original debug and warn logged. Overridden debug and warn filtered. Overridden error logged.
	assert.Equal(t, 1, debugCount, "only 'original debug' should appear")
	assert.Equal(t, 1, warnCount, "only 'original warn' should appear")
	assert.Equal(t, 1, errorCount, "only 'overridden error' should appear")
}

func TestApplyLogLevelOverride(t *testing.T) {
	// Setup base TelemetrySettings with a debug-level logger
	core, obs := observer.New(zapcore.DebugLevel)
	baseLogger := zap.New(core)
	baseSettings := component.TelemetrySettings{
		Logger: baseLogger,
	}

	// Override map sets warn level for component "nop"
	warnLevel := zapcore.WarnLevel
	overrides := map[component.ID]*zapcore.Level{
		component.NewID(component.MustNewType("nop")): &warnLevel,
	}

	nopID := component.NewID(component.MustNewType("nop"))
	otherID := component.NewID(component.MustNewType("other"))

	// nop should get overridden (warn level -> debug filtered, warn passed)
	nopSettings := componentattribute.ApplyLogLevelOverride(baseSettings, overrides, nopID)
	nopSettings.Logger.Debug("nop debug")
	nopSettings.Logger.Warn("nop warn")

	// other should NOT get overridden (falls through to default debug)
	otherSettings := componentattribute.ApplyLogLevelOverride(baseSettings, overrides, otherID)
	otherSettings.Logger.Debug("other debug")
	otherSettings.Logger.Warn("other warn")

	logs := obs.TakeAll()
	var debugCount, warnCount int
	for _, l := range logs {
		switch l.Level {
		case zapcore.DebugLevel:
			debugCount++
		case zapcore.WarnLevel:
			warnCount++
		}
	}

	// nop debug filtered, nop warn logged; other debug and warn both logged
	assert.Equal(t, 1, debugCount, "only 'other debug' should appear (nop debug filtered)")
	assert.Equal(t, 2, warnCount, "both warns should appear")
}

func TestApplyLogLevelOverride_NilOverrides(t *testing.T) {
	core, obs := observer.New(zapcore.DebugLevel)
	baseLogger := zap.New(core)
	baseSettings := component.TelemetrySettings{
		Logger: baseLogger,
	}

	// Nil overrides should leave settings unchanged
	settings := componentattribute.ApplyLogLevelOverride(baseSettings, nil, component.NewID(component.MustNewType("nop")))
	settings.Logger.Debug("debug msg")

	logs := obs.TakeAll()
	require.Len(t, logs, 1)
	assert.Equal(t, zapcore.DebugLevel, logs[0].Level)
}
