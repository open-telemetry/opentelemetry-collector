// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	semconv118 "go.opentelemetry.io/otel/semconv/v1.18.0"

	"go.opentelemetry.io/collector/config/configtelemetry"
)

// TODO views should be passed in as options to Factory.CreateMeterProvider
// from the service package, so they are independent of the telemetry
// implementation. This isn't workable at present as sdk.MeterProvider()
// does not allow options.

func dropViewOption(selector *config.ViewSelector) config.View {
	return config.View{
		Selector: selector,
		Stream: &config.ViewStream{
			Aggregation: &config.ViewStreamAggregation{
				Drop: config.ViewStreamAggregationDrop{},
			},
		},
	}
}

func configureViews(level configtelemetry.Level) []config.View {
	views := []config.View{}

	if level < configtelemetry.LevelDetailed {
		// Drop all otelhttp and otelgrpc metrics if the level is not detailed.
		views = append(views,
			dropViewOption(&config.ViewSelector{
				MeterName: ptr("go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"),
			}),
			dropViewOption(&config.ViewSelector{
				MeterName: ptr("go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"),
			}),
		)
	}

	// Make sure to add the AttributeKeys view after the AggregationDrop view:
	// Only the first view outputting a given metric identity is actually used, so placing the
	// AttributeKeys view first would never drop the metrics regadless of level.
	if disableHighCardinalityMetricsFeatureGate.IsEnabled() {
		views = append(views, []config.View{
			{
				Selector: &config.ViewSelector{
					MeterName: ptr("go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"),
				},
				Stream: &config.ViewStream{
					AttributeKeys: &config.IncludeExclude{
						Excluded: []string{
							string(semconv118.NetSockPeerAddrKey),
							string(semconv118.NetSockPeerPortKey),
							string(semconv118.NetSockPeerNameKey),
						},
					},
				},
			},
			{
				Selector: &config.ViewSelector{
					MeterName: ptr("go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"),
				},
				Stream: &config.ViewStream{
					AttributeKeys: &config.IncludeExclude{
						Excluded: []string{
							string(semconv118.NetHostNameKey),
							string(semconv118.NetHostPortKey),
						},
					},
				},
			},
		}...)
	}

	// otel-arrow library metrics
	// See https://github.com/open-telemetry/otel-arrow/blob/c39257/pkg/otel/arrow_record/consumer.go#L174-L176
	if level < configtelemetry.LevelNormal {
		scope := ptr("otel-arrow/pkg/otel/arrow_record")
		views = append(views,
			dropViewOption(&config.ViewSelector{
				MeterName:      scope,
				InstrumentName: ptr("arrow_batch_records"),
			}),
			dropViewOption(&config.ViewSelector{
				MeterName:      scope,
				InstrumentName: ptr("arrow_schema_resets"),
			}),
			dropViewOption(&config.ViewSelector{
				MeterName:      scope,
				InstrumentName: ptr("arrow_memory_inuse"),
			}),
		)
	}

	// contrib's internal/otelarrow/netstats metrics
	// See
	// - https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/a25f05/internal/otelarrow/netstats/netstats.go#L130
	// - https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/a25f05/internal/otelarrow/netstats/netstats.go#L165
	if level < configtelemetry.LevelDetailed {
		scope := ptr("github.com/open-telemetry/opentelemetry-collector-contrib/internal/otelarrow/netstats")

		views = append(views,
			// Compressed size metrics.
			dropViewOption(&config.ViewSelector{
				MeterName:      scope,
				InstrumentName: ptr("otelcol_*_compressed_size"),
			}),
			dropViewOption(&config.ViewSelector{
				MeterName:      scope,
				InstrumentName: ptr("otelcol_*_compressed_size"),
			}),

			// makeRecvMetrics for exporters.
			dropViewOption(&config.ViewSelector{
				MeterName:      scope,
				InstrumentName: ptr("otelcol_exporter_recv"),
			}),
			dropViewOption(&config.ViewSelector{
				MeterName:      scope,
				InstrumentName: ptr("otelcol_exporter_recv_wire"),
			}),

			// makeSentMetrics for receivers.
			dropViewOption(&config.ViewSelector{
				MeterName:      scope,
				InstrumentName: ptr("otelcol_receiver_sent"),
			}),
			dropViewOption(&config.ViewSelector{
				MeterName:      scope,
				InstrumentName: ptr("otelcol_receiver_sent_wire"),
			}),
		)
	}

	// Batch processor metrics
	scope := ptr("go.opentelemetry.io/collector/processor/batchprocessor")
	if level < configtelemetry.LevelNormal {
		views = append(views, dropViewOption(&config.ViewSelector{
			MeterName: scope,
		}))
	} else if level < configtelemetry.LevelDetailed {
		views = append(views, dropViewOption(&config.ViewSelector{
			MeterName:      scope,
			InstrumentName: ptr("otelcol_processor_batch_batch_send_size_bytes"),
		}))
	}

	// Internal graph metrics
	graphScope := ptr("go.opentelemetry.io/collector/service")
	if level < configtelemetry.LevelDetailed {
		views = append(views, dropViewOption(&config.ViewSelector{
			MeterName:      graphScope,
			InstrumentName: ptr("otelcol.*.consumed.size"),
		}))
		views = append(views, dropViewOption(&config.ViewSelector{
			MeterName:      graphScope,
			InstrumentName: ptr("otelcol.*.produced.size"),
		}))
	}

	return views
}

func ptr[T any](v T) *T {
	return &v
}
