// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute // import "go.opentelemetry.io/collector/internal/telemetry/componentattribute"

import (
	"errors"
	"reflect"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Interface for Zap cores that support setting and resetting a set of component attributes.
//
// There are three wrappers that implement this interface:
//
//   - [NewConsoleCoreWithAttributes] injects component attributes as Zap fields.
//
//     This is used for the Collector's console output.
//
//   - [NewOTelTeeCoreWithAttributes] copies logs to a [log.LoggerProvider] using [otelzap]. For the
//     copied logs, component attributes are injected as instrumentation scope attributes.
//
//     This is used when service::telemetry::logs::processors is configured.
//
//   - [NewWrapperCoreWithAttributes] applies a wrapper function to a core, similar to
//     [zap.WrapCore]. It allows setting component attributes on the inner core and reapplying the
//     wrapper function when needed.
//
//     This is used when adding [zapcore.NewSamplerWithOptions] to our logger stack.
type coreWithAttributes interface {
	zapcore.Core
	withAttributeSet(attribute.Set) zapcore.Core
}

// Tries setting the component attribute set for a Zap core.
//
// Does nothing if the core does not implement [coreWithAttributes].
func tryWithAttributeSet(c zapcore.Core, attrs attribute.Set) zapcore.Core {
	if cwa, ok := c.(coreWithAttributes); ok {
		return cwa.withAttributeSet(attrs)
	}
	zap.New(c).Debug("Logger core does not support injecting component attributes")
	return c
}

type consoleCoreWithAttributes struct {
	zapcore.Core
	from zapcore.Core
}

var _ coreWithAttributes = (*consoleCoreWithAttributes)(nil)

// NewConsoleCoreWithAttributes wraps a Zap core in order to inject component attributes as Zap fields.
//
// This is used for the Collector's console output.
func NewConsoleCoreWithAttributes(c zapcore.Core, attrs attribute.Set) zapcore.Core {
	var fields []zap.Field
	for _, kv := range attrs.ToSlice() {
		fields = append(fields, zap.String(string(kv.Key), kv.Value.AsString()))
	}
	return &consoleCoreWithAttributes{
		Core: c.With(fields),
		from: c,
	}
}

func (ccwa *consoleCoreWithAttributes) withAttributeSet(attrs attribute.Set) zapcore.Core {
	return NewConsoleCoreWithAttributes(ccwa.from, attrs)
}

type otelTeeCoreWithAttributes struct {
	sourceCore zapcore.Core
	otelCore   zapcore.Core
	lp         log.LoggerProvider
	scopeName  string
}

var _ coreWithAttributes = (*otelTeeCoreWithAttributes)(nil)

// NewOTelTeeCoreWithAttributes wraps a Zap core in order to copy logs to a [log.LoggerProvider] using [otelzap].
// For the copied logs, component attributes are injected as instrumentation scope attributes.
//
// Note that we intentionally do not use zapcore.NewTee here, because it will simply duplicate all log entries
// to each core. We only want to copy logs to the [log.LoggerProvider] if the provided core accepts the log.
func NewOTelTeeCoreWithAttributes(core zapcore.Core, lp log.LoggerProvider, scopeName string, attrs attribute.Set) zapcore.Core {
	otelCore := otelzap.NewCore(
		scopeName,
		otelzap.WithLoggerProvider(lp),
		otelzap.WithAttributes(attrs.ToSlice()...),
	)
	return &otelTeeCoreWithAttributes{
		sourceCore: core,
		otelCore:   otelCore,
		lp:         lp,
		scopeName:  scopeName,
	}
}

func (ocwa *otelTeeCoreWithAttributes) withAttributeSet(attrs attribute.Set) zapcore.Core {
	return NewOTelTeeCoreWithAttributes(
		tryWithAttributeSet(ocwa.sourceCore, attrs),
		ocwa.lp, ocwa.scopeName, attrs,
	)
}

func (ocwa *otelTeeCoreWithAttributes) With(fields []zapcore.Field) zapcore.Core {
	sourceCoreWith := ocwa.sourceCore.With(fields)
	otelCoreWith := ocwa.otelCore.With(fields)
	return &otelTeeCoreWithAttributes{
		sourceCore: sourceCoreWith,
		otelCore:   otelCoreWith,
		lp:         ocwa.lp,
		scopeName:  ocwa.scopeName,
	}
}

func (ocwa *otelTeeCoreWithAttributes) Enabled(level zapcore.Level) bool {
	return ocwa.sourceCore.Enabled(level)
}

func (ocwa *otelTeeCoreWithAttributes) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	ce = ocwa.sourceCore.Check(entry, ce)
	if ce != nil {
		// Only log to the otelzap core if the input core accepted the log entry.
		ce = ce.AddCore(entry, ocwa.otelCore)
	}
	return ce
}

func (ocwa *otelTeeCoreWithAttributes) Write(zapcore.Entry, []zapcore.Field) error {
	// This wrapper's Write method should not be called. The underlying sourceCore
	// and otelCore will add themselves to the CheckedEntry, which will result in their
	// Write methods being called directly.
	return errors.New("Write should not be called directly")
}

func (ocwa *otelTeeCoreWithAttributes) Sync() error {
	err := ocwa.sourceCore.Sync()
	return multierr.Append(err, ocwa.otelCore.Sync())
}

type samplerCoreWithAttributes struct {
	zapcore.Core
	from zapcore.Core
}

var _ coreWithAttributes = (*samplerCoreWithAttributes)(nil)

func NewSamplerCoreWithAttributes(inner zapcore.Core, tick time.Duration, first int, thereafter int) zapcore.Core {
	return &samplerCoreWithAttributes{
		Core: zapcore.NewSamplerWithOptions(inner, tick, first, thereafter),
		from: inner,
	}
}

func checkSamplerType(ty reflect.Type) bool {
	if ty.Kind() != reflect.Pointer {
		return false
	}
	ty = ty.Elem()
	if ty.Kind() != reflect.Struct {
		return false
	}
	innerField, ok := ty.FieldByName("Core")
	if !ok {
		return false
	}
	return reflect.TypeFor[zapcore.Core]().AssignableTo(innerField.Type)
}

func (ssc *samplerCoreWithAttributes) withAttributeSet(attrs attribute.Set) zapcore.Core {
	newInner := tryWithAttributeSet(ssc.from, attrs)

	// Relevant Zap code: https://github.com/uber-go/zap/blob/fcf8ee58669e358bbd6460bef5c2ee7a53c0803a/zapcore/sampler.go#L168
	// We need to create a new Zap sampler core with the same settings but with a new inner core,
	// while reusing the very RAM-intensive `counters` data structure.
	// The `With` method does something similar, but it only replaces pre-set fields, not the Core.
	// However, we can use `reflect` to accomplish this.
	// This hack can be removed once Zap supports this use case.
	// Tracking issue: https://github.com/uber-go/zap/issues/1498
	val1 := reflect.ValueOf(ssc.Core)
	if !checkSamplerType(val1.Type()) { // To avoid a more esoteric panic message below
		panic("Unexpected Zap sampler type; see github.com/open-telemetry/opentelemetry-collector/issues/13014")
	}
	val2 := reflect.New(val1.Type().Elem())                        // core2 := new(sampler)
	val2.Elem().Set(val1.Elem())                                   // *core2 = *core1
	val2.Elem().FieldByName("Core").Set(reflect.ValueOf(newInner)) // core2.Core = newInner
	newSampler := val2.Interface().(zapcore.Core)

	return samplerCoreWithAttributes{
		Core: newSampler,
		from: newInner,
	}
}

// ZapLoggerWithAttributes creates a Zap Logger with a new set of injected component attributes.
func ZapLoggerWithAttributes(logger *zap.Logger, attrs attribute.Set) *zap.Logger {
	return logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return tryWithAttributeSet(c, attrs)
	}))
}
