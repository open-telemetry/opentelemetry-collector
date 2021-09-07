# Changelog

## Unreleased

## v0.34.0 Beta

## 🛑 Breaking changes 🛑

- Artifacts are no longer published in this repository, check [here](https://github.com/open-telemetry/opentelemetry-collector-releases) (#3941)
- Remove deprecated `tracetranslator.AttributeValueToString` and `tracetranslator.AttributeMapToMap` (#3873)
- Change semantic conventions for status (code, msg) as per specifications (#3872)
- Add `pdata.NewTimestampFromTime`, deprecate `pdata.TimestampFromTime` (#3868)
- Add `pdata.NewAttributeMapFromMap`, deprecate `pdata.AttributeMap.InitFromMap` (#3936)
- Move `fileexporter` to contrib (#3474)
- Move `jaegerexporter` to contrib (#3474)
- Move `kafkaexporter` to contrib (#3474)
- Move `opencensusexporter` to contrib (#3474)
- Move `prometheusexporter` to contrib (#3474)
- Move `prometheusremotewriteexporter` to contrib (#3474)
- Move `zipkinexporter` to contrib (#3474)
- Move `attributeprocessor` to contrib (#3474)
- Move `filterprocessor` to contrib (#3474)
- Move `probabilisticsamplerprocessor` to contrib (#3474)
- Move `resourceprocessor` to contrib (#3474)
- Move `spanprocessor` to contrib (#3474)
- Move `hostmetricsreceiver` to contrib (#3474)
- Move `jaegerreceiver` to contrib (#3474)
- Move `kafkareceiver` to contrib (#3474)
- Move `opencensusreceiver` to contrib (#3474)
- Move `prometheusreceiver` to contrib (#3474)
- Move `zipkinreceiver` to contrib (#3474)
- Move `bearertokenauthextension` to contrib (#3474)
- Move `healthcheckextension` to contrib (#3474)
- Move `oidcauthextension` to contrib (#3474)
- Move `pprofextension` to contrib (#3474)
- Move `translator/internaldata` to contrib (#3474)
- Move `translator/trace/jaeger` to contrib (#3474)
- Move `translator/trace/zipkin` to contrib (#3474)
- Move `testbed` to contrib (#3474)
- Move `exporter/exporterhelper/resource_to_telemetry` to contrib (#3474)
- Move `processor/processorhelper/attraction` to contrib (#3474)
- Move `translator/conventions` to `model/semconv` (#3901)

## v0.33.0 Beta

## 🛑 Breaking changes 🛑

- Rename `configloader` interface to `configunmarshaler` (#3774)
- Remove `LabelsMap` from all the metrics points (#3706)
- Update generated K8S attribute labels to fix capitalization (#3823) 

## 💡 Enhancements 💡

- Collector has now full support for metrics proto v0.9.0.

## v0.32.0 Beta

This release is marked as "bad" since the metrics pipelines will produce bad data.

- See https://github.com/open-telemetry/opentelemetry-collector/issues/3824

## 🛑 Breaking changes 🛑

- Rename `CustomUnmarshable` interface to `Unmarshallable` (#3774)

## 💡 Enhancements 💡

- Change default OTLP/HTTP port number from 55681 to 4318 (#3743)
- Update OTLP proto to v0.9.0 (#3740)
  - Remove `SetValue`/`Value` func for `NumberDataPoint`/`Exemplar` (#3730)
  - Remove `IntGauge`/`IntSum`from pdata (#3731)
  - Remove `IntDataPoint` from pdata (#3735)
  - Add support for `Bytes` attribute type (#3756)
  - Add `SchemaUrl` field (#3759)
  - Add `Attributes` to `NumberDataPoint`, `HistogramDataPoint`, `SummaryDataPoint` (#3761)
- `conventions` translator: Replace with conventions generated from spec v1.5.0 (#3494)
- `prometheus` receiver: Add `ToMetricPdata` method (#3695)
- Make configsource `Watchable` an optional interface (#3792)
- `obsreport` exporter: Change to accept `ExporterCreateSettings` (#3789)

## 🧰 Bug fixes 🧰

- `configgrpc`: Use chained interceptors in the gRPC server (#3744)
- `prometheus` receiver: Use actual interval startTimeMs for cumulative types (#3694)
- `jaeger` translator: Fix bug that could generate empty proto spans (#3808)

## v0.31.0 Beta

## 🛑 Breaking changes 🛑

- Remove Resize() from pdata slice APIs (#3675)
- Remove the ballast allocation when `mem-ballast-size-mib` is set in command line (#3626)
  - Use [`ballast extension`](./extension/ballastextension/README.md) to set memory ballast instead.
- Rename `DoubleDataPoint` to `NumberDataPoint` (#3633)
- Remove `IntHistogram` (#3676)

## 💡 Enhancements 💡

- Update to OTLP 0.8.0:
  - Translate `IntHistogram` to `Histogram` in `otlp_wrappers` (#3676)
  - Translate `IntGauge` to `Gauge` in `otlp_wrappers` (#3619)
  - Translate `IntSum` to `Sum` in `otlp_wrappers` (#3621)
  - Update `NumberDataPoint` to support `DoubleVal` and `IntVal` (#3689)
  - Update `Exemplar` to use `oneOfPrimitiveValue` (#3699)
  - Remove `IntExemplar` and `IntExemplarSlice` from `pdata` (#3705)
  - Mark `IntGauge`/`IntSum`/`IntDataPoint` as deprecated (#3707)
  - Remove `IntGauge`/`IntSum` from `batchprocessor` (#3718)
  - `prometheusremotewrite` exporter: Convert to new Number metrics (#3714)
  - `prometheus` receiver: Convert to new Number metrics (#3716)
  - `prometheus` exporter: Convert to new Number metrics (#3709)
  - `hostmetrics` receiver: Convert to new Number metrics (#3710)
  - `opencensus`: Convert to new Number metrics (#3708)
  - `scraperhelper` receiver: Convert to new Number metrics (#3717)
  - `testbed`: Convert to new Number metrics (#3719)
  - `expoerterhelper`: Convert `resourcetolabel` to new Number metrics (#3723)
- `configauth`: Prepare auth API to return a context (#3618)
- `pdata`:
  - Implement `Equal()` for map-valued `AttributeValues` (#3612)
  - Add `[Type]Slice.Sort(func)` to sort slices (#3671)
- `memorylimiter`:
  - Add validation on ballast size between `memorylimiter` and `ballastextension` (#3532)
  - Access Ballast extension via `Host.GetExtensions` (#3634)
- `prometheusremotewrite` exporter: Add a WAL implementation without wiring up (#3597)
- `prometheus` receiver: Add `metricGroup.toDistributionPoint` pdata conversion (#3667)
- Use `ComponentID` as identifier instead of config (#3696)
- `zpages`: Move config validation from factory to `Validate` (#3697)
- Enable `tracez` z-pages from otel-go, disable opencensus (#3698)
- Convert temporality and monotonicity for deprecated sums (#3729)

## 🧰 Bug fixes 🧰

- `otlpexporter`: Allow endpoint to be configured with a scheme of `http` or `https` (#3575)
- Handle errors when reloading the collector service (#3615)
- Do not report fatal error when `cmux.ErrServerClosed` (#3703)
- Fix bool attribute equality in `pdata` (#3688)

## v0.30.0 Beta

## 🛑 Breaking changes 🛑

- Rename `pdata.DoubleSum` to `pdata.Sum` (#3583)
- Rename `pdata.DoubleGauge` to `pdata.Gauge` (#3599)
- Migrated `pdata` to a dedicated package (#3483)
- Change Marshaler/Unmarshaler to be consistent with other interfaces (#3502)
- Remove consumer/simple package (#3438)
- Remove unnecessary interfaces from pdata (#3506)
- zipkinv1 implement directly Unmarshaler interface (#3504)
- zipkinv2 implement directly Marshaler/Unmarshaler interface (#3505)
- Change exporterhelper to accept ExporterCreateSettings instead of just logger (#3569)
- Deprecate Resize() from pdata slice APIs (#3573) 
- Use Func pattern in processorhelper, consistent with others (#3570)

## 💡 Enhancements 💡

- Update OTLP to v0.8.0 (#3572)
- Migrate from OpenCensus to OpenTelemetry for internal tracing (#3567) 
- Move internal/pdatagrpc to model/otlpgrpc (#3507) 
- Move internal/otlp to model/otlp (#3508)
- Create http Server via Config, enable cors and decompression (#3513)
- Allow users to set min and max TLS versions (#3591)
- Support setting ballast size in percentage of total Mem in ballast extension (#3456)
- Publish go.opentelemetry.io/collector/model as a separate module (#3530)
- Pass a TracerProvider via construct settings to all the components (#3592) 
- Make graceful shutdown optional (#3577)

## 🧰 Bug fixes 🧰

- `scraperhelper`: Include the scraper name in log messages (#3487)
- `scraperhelper`: fix case when returned pdata is empty (#3520) 
- Record the correct number of points not metrics in Kafka receiver (#3553) 
- Validate the Prometheus configuration (#3589) 

## v0.29.0 Beta

## 🛑 Breaking changes 🛑

- Rename `service.Application` to `service.Collector` (#3268)
- Provide case sensitivity in config yaml mappings by using Koanf instead of Viper (#3337)
- Move zipkin constants to an internal package (#3431)
- Disallow renaming metrics using metric relabel configs (#3410)
- Move cgroup and iruntime utils from memory_limiter to internal folder (#3448)
- Move model pdata interfaces to pdata, expose them publicly (#3455)

## 💡 Enhancements 💡

- Change obsreport helpers for scraper to use the same pattern as Processor/Exporter (#3327)
- Convert `otlptext` to implement Marshaler interfaces (#3366)
- Add encoder/decoder and marshaler/unmarshaler for OTLP protobuf (#3401)
- Use the new marshaler/unmarshaler in `kafka` exporter (#3403)
- Convert `zipkinv2` to to/from translator interfaces (#3409)
- `zipkinv1`: Move to translator and encoders interfaces (#3419)
- Use the new marshaler/unmarshaler in `kafka` receiver #3402
- Change `oltp` receiver to use the new unmarshaler, avoid grpc-gateway dependency (#3406)
- Use the new Marshaler in the `otlphttp` exporter (#3433)
- Add grpc response struct for all signals instead of returning interface in `otlp` receiver/exporter (#3437)
- `zipkinv2`: Add encoders, decoders, marshalers (#3426)
- `scrapererror` receiver: Return concrete error type (#3360)
- `kafka` receiver: Add metrics support (#3452)
- `prometheus` receiver:
  - Add store to track stale metrics (#3414)
  - Add `up` and `scrape_xxxx` internal metrics (#3116)

## 🧰 Bug fixes 🧰

- `prometheus` receiver:
  - Reject datapoints with duplicate label keys (#3408)
  - Scrapers are not stopped when receiver is shutdown (#3450)
- `prometheusremotewrite` exporter: Adjust default retry settings (#3416)
- `hostmetrics` receiver: Fix missing startTimestamp for `processes` scraper (#3461)

## v0.28.0 Beta

## 🛑 Breaking changes 🛑

- Remove unused logstest package (#3222)
- Introduce `AppSettings` instead of `Parameters` (#3163)
- Remove unused testutil.TempSocketName (#3291)
- Move BigEndian helper functions in `tracetranslator` to an internal package.(#3298)
- Rename `configtest.LoadConfigFile` to `configtest.LoadConfigAndValidate` (#3306)
- Replace `ExtensionCreateParams` with `ExtensionCreateSettings` (#3294)
- Replace `ProcessorCreateParams` with `ProcessorCreateSettings`. (#3181)
- Replace `ExporterCreateParams` with `ExporterCreateSettings` (#3164)
- Replace `ReceiverCreateParams` with `ReceiverCreateSettings`. (#3167)
- Change `batchprocessor` logic to limit data points rather than metrics (#3141)
- Rename `PrwExporter` to `PRWExporter` and `NewPrwExporter` to `NewPRWExporter` (#3246)
- Avoid exposing OpenCensus reference in public APIs (#3253)
- Move `config.Parser` to `configparser.Parser` (#3304)
- Remove deprecated funcs inside the obsreceiver (#3314)
- Remove `obsreport.GRPCServerWithObservabilityEnabled`, enable observability in config (#3315)
- Remove `obsreport.ProcessorMetricViews`, use `BuildProcessorCustomMetricName` where needed (#3316)
- Remove "Receive" from `obsreport.Receiver` funcs (#3326)
- Remove "Export" from `obsreport.Exporter` funcs (#3333)
- Hide unnecessary public struct `obsreport.StartReceiveOptions` (#3353)
- Avoid exposing internal implementation public in OC/OTEL receivers (#3355)
- Updated configgrpc `ToDialOptions` and confighttp `ToClient` apis to take extensions configuration map (#3340)
- Remove `GenerateSequentialTraceID` and `GenerateSequentialSpanIDin` functions in testbed (#3390)
- Change "grpc" to "GRPC" in configauth function/type names (#3285)

## 💡 Enhancements 💡

- Add `doc.go` files to the consumer package and its subpackages (#3270)
- Improve documentation of consumer package and subpackages (#3269, #3361)
- Automate triggering of doc-update on release (#3234)
- Enable Dependabot for Github Actions (#3312)
- Remove the proto dependency in `goldendataset` for traces (#3322)
- Add telemetry for dropped data due to exporter sending queue overflow (#3328)
- Add initial implementation of `pdatagrcp` (#3231)
- Change receiver obsreport helpers pattern to match the Processor/Exporter (#3227)
- Add model translation and encoding interfaces (#3200)
- Add otlpjson as a serializer implementation (#3238)
- `prometheus` receiver:
  - Add `createNodeAndResourcePdata` for Prometheus->OTLP pdata (#3139)
  - Direct metricfamily Prometheus->OTLP (#3145)
- Add `componenttest.NewNop*CreateSettings` to simplify tests (#3375)
- Add support for markdown generation (#3100)
- Refactor components for the Client Authentication Extensions (#3287)

## 🧰 Bug fixes 🧰

- Use dedicated `zapcore.Core` for Windows service (#3147)
- Hook up start and shutdown functions in fileexporter (#3260)
- Fix oc to pdata translation for sum non-monotonic cumulative (#3272)
- Fix `timeseriesSignature` in prometheus receiver (#3310)

## v0.27.0 Beta

## 🛑 Breaking changes 🛑

- Change `Marshal` signatures in kafkaexporter's Marshalers to directly convert pdata to `sarama.ProducerMessage` (#3162)
- Remove `tracetranslator.DetermineValueType`, only used internally by Zipkin (#3114)
- Remove OpenCensus conventions, should not be used (#3113)
- Remove Zipkin specific translation constants, move to internal (#3112)
- Remove `tracetranslator.TagHTTPStatusCode`, use `conventions.AttributeHTTPStatusCode` (#3111)
- Remove OpenCensus status constants and transformation (#3110)
- Remove `tracetranslator.AttributeArrayToSlice`, not used in core or contrib (#3109)
- Remove `internaldata.MetricsData`, same APIs as for traces (#3156)
- Rename `config.IDFromString` to `NewIDFromString`, remove `MustIDFromString` (#3177)
- Move consumerfanout package to internal (#3207)
- Canonicalize enum names in pdata. Fix usage of uppercase names (#3208)

## 💡 Enhancements 💡

- Use `config.ComponentID` for obsreport receiver/scraper (#3098)
- Add initial implementation of the consumerhelper (#3146)
- Add Collector version to Prometheus Remote Write Exporter user-agent header (#3094)
- Refactor processorhelper to use consumerhelper, split by signal type (#3180)
- Use consumerhelper for exporterhelper, add WithCapabilities (#3186)
- Set capabilities for all core exporters, remove unnecessary funcs (#3190)
- Add an internal sharedcomponent to be shared by receivers with shared resources (#3198)
- Allow users to configure the Prometheus remote write queue (#3046)
- Mark internaldata traces translation as deprecated for external usage (#3176)

## 🧰 Bug fixes 🧰

- Fix Prometheus receiver metric start time and reset determination logic. (#3047)
  - The receiver will no longer drop the first sample for `counter`, `summary`, and `histogram` metrics.
- The Prometheus remote write exporter will no longer force `counter` metrics to have a `_total` suffix. (#2993)
- Remove locking from jaeger receiver start and stop processes (#3070)
- Fix batch processor metrics reorder, improve performance (#3034)
- Fix batch processor traces reorder, improve performance (#3107)
- Fix batch processor logs reorder, improve performance (#3125)
- Avoid one unnecessary allocation in grpc OTLP exporter (#3122)
- `batch` processor: Validate that batch config max size is greater than send size (#3126)
- Add capabilities to consumer, remove from processor (#2770)
- Remove internal protos usage in Prometheusremotewrite exporter (#3184)
- `prometheus` receiver: Honor Prometheus external labels (#3127)
- Validate that remote write queue settings are not negative (#3213)

## v0.26.0 Beta

## 🛑 Breaking changes 🛑

- Change `With*Unmarshallers` signatures in Kafka exporter/receiver (#2973)
- Rename `marshall` to `marshal` in all the occurrences (#2977)
- Remove `componenterror.ErrAlreadyStarted` and `componenterror.ErrAlreadyStopped`, components should not protect against this, Service will start/stop once.
- Rename `ApplicationStartInfo` to `BuildInfo`
- Rename `ApplicationStartInfo.ExeName` to `BuildInfo.Command`
- Rename `ApplicationStartInfo.LongName` to `BuildInfo.Description`

## 💡 Enhancements 💡

- `kafka` exporter: Add logs support (#2943)
- Add AppendEmpty and deprecate Append for slices (#2970)
- Update mdatagen to create factories of init instead of new (#2978)
- `zipkin` receiver: Reduce the judgment of zipkin v1 version (#2990)
- Custom authenticator logic to accept a `component.Host` which will extract the authenticator to use based on a new authenticator name property (#2767)
- `prometheusremotewrite` exporter: Add `resource_to_telemetry_conversion` config option (#3031)
- `logging` exporter: Extract OTLP text logging (#3082)
- Format timestamps as strings instead of int in otlptext output (#3088)
- Add darwin arm64 build (#3090)

## 🧰 Bug fixes 🧰

- Fix Jaeger receiver to honor TLS Settings (#2866)
- `zipkin` translator: Handle missing starttime case for zipkin json v2 format spans (#2506)
- `prometheus` exporter: Fix OTEL resource label drops (#2899)
- `prometheusremotewrite` exporter:
  - Enable the queue internally (#2974)
  - Don't drop instance and job labels (#2979)
- `jaeger` receiver: Wait for server goroutines exit on shutdown (#2985)
- `logging` exporter: Ignore invalid handle on close (#2994)
- Fix service zpages (#2996)
- `batch` processor: Fix to avoid reordering and send max size (#3029)


## v0.25.0 Beta

## 🛑 Breaking changes 🛑

- Rename ForEach (in pdata) with Range to be consistent with sync.Map (#2931)
- Rename `componenthelper.Start` to `componenthelper.StartFunc` (#2880)
- Rename `componenthelper.Stop` to `componenthelper.StopFunc` (#2880)
- Remove `exporterheleper.WithCustomUnmarshaler`, `processorheleper.WithCustomUnmarshaler`, `receiverheleper.WithCustomUnmarshaler`, `extensionheleper.WithCustomUnmarshaler`, implement `config.CustomUnmarshaler` interface instead (#2867)
- Remove `component.CustomUnmarshaler` implement `config.CustomUnmarshaler` interface instead (#2867)
- Remove `testutil.HostPortFromAddr`, users can write their own parsing helper (#2919)
- Remove `configparser.DecodeTypeAndName`, use `config.IDFromString` (#2869)
- Remove `config.NewViper`, users should use `config.NewParser` (#2917)
- Remove `testutil.WaitFor`, use `testify.Eventually` helper if needed (#2920)
- Remove testutil.WaitForPort, users can use testify.Eventually (#2926)
- Rename `processorhelper.NewTraceProcessor` to `processorhelper.NewTracesProcessor` (#2935)
- Rename `exporterhelper.NewTraceExporter` to `exporterhelper.NewTracesExporter` (#2937)
- Remove InitEmptyWithCapacity, add EnsureCapacity and Clear (#2845)
- Rename traces methods/objects to include Traces in Kafka receiver (#2966)

## 💡 Enhancements 💡

- Add `validatable` interface with `Validate()` to all `config.<component>` (#2898)
  - add the empty `Validate()` implementation for all component configs
- **Experimental**: Add a config source manager that wraps the interaction with config sources (#2857, #2903, #2948)
- `kafka` exporter: Key jaeger messages on traceid (#2855)
- `scraperhelper`: Don't try to count metrics if scraper returns an error (#2902)
- Extract ConfigFactory in a ParserProvider interface (#2868)
- `prometheus` exporter: Allows Summary metrics to be exported to Prometheus (#2900)
- `prometheus` receiver: Optimize `dpgSignature` function (#2945)
- `kafka` receiver: Add logs support (#2944)

## 🧰 Bug fixes 🧰

- `prometheus` receiver:
  - Treat Summary and Histogram metrics without "_sum" counter as valid metric (#2812)
  - Add `job` and `instance` as well-known labels (#2897)
- `prometheusremotewrite` exporter:
  - Sort Sample by Timestamp to avoid out of order errors (#2941)
  - Remove incompatible queued retry (#2951)
- `kafka` receiver: Fix data race with batchprocessor (#2957)
- `jaeger` receiver: Jaeger agent should not report ErrServerClosed (#2965)

## v0.24.0 Beta

## 🛑 Breaking changes 🛑

- Remove legacy internal metrics for memorylimiter processor, `spans_dropped` and `trace_batches_dropped` (#2841)
  - For `spans_dropped` use `processor/refused_spans` with `processor=memorylimiter`
- Rename pdata.*.[Start|End]Time to pdata.*.[Start|End]Timestamp (#2847)
- Rename pdata.DoubleExemplar to pdata.Exemplar (#2804)
- Rename pdata.DoubleHistogram to pdata.Histogram (#2797)
- Rename pdata.DoubleSummary to pdata.Summary (#2774)
- Refactor `consumererror` package (#2768)
  - Remove `PartialError` type in favor of signal-specific types
  - Rename `CombineErrors()` to `Combine()`
- Refactor `componenthelper` package (#2778)
  - Remove `ComponentSettings` and `DefaultComponentSettings()`
  - Rename `NewComponent()` to `New()`
- obsReport.NewExporter accepts a settings struct (#2668)
- Remove ErrorWaitingHost from `componenttest` (#2582)
- Move `config.Load` to `configparser.Load` (#2796)
- Remove `configtest.NewViperFromYamlFile()`, use `config.Parser.NewParserFromFile()` (#2806)
- Remove `config.ViperSubExact()`, use `config.Parser.Sub()` (#2806)
- Update LoadReceiver signature to remove unused params (#2823)
- Move `configerror.ErrDataTypeIsNotSupported` to `componenterror.ErrDataTypeIsNotSupported` (#2886)
- Rename`CreateTraceExporter` type to `CreateTracesExporter` in `exporterhelper` (#2779)
- Move `fluentbit` extension to contrib (#2795)
- Move `configmodels` to `config` (#2808)
- Move `fluentforward` receiver to contrib (#2723)

## 💡 Enhancements 💡

- `batch` processor: - Support max batch size for logs (#2736)
- Use `Endpoint` for health check extension (#2782)
- Use `confignet.TCPAddr` for `pprof` and `zpages` extensions (#2829)
- Deprecate `consumetest.New[${SIGNAL}]Nop` in favor of `consumetest.NewNop` (#2878)
- Deprecate `consumetest.New[${SIGNAL}]Err` in favor of `consumetest.NewErr` (#2878)
- Add watcher to values retrieved via config sources (#2803)
- Updates for cloud semantic conventions (#2809)
  - `cloud.infrastructure_service` -> `cloud.platform`
  - `cloud.zone` -> `cloud.availability_zone`
- Add systemd environment file for deb/rpm packages (#2822)
- Add validate interface in `configmodels` to force each component do configuration validation (#2802, #2856)
- Add `aws.ecs.task.revision` to semantic conventions list (#2816)
- Set unprivileged user to container image (#2838)
- Add New funcs for extension, exporter, processor config settings (#2872)
- Report metric about current size of the exporter retry queue (#2858)
- Allow adding new signals in `ProcessorFactory` by forcing everyone to embed `BaseProcessorFactory` (#2885)

## 🧰 Bug fixes 🧰

- `pdata.TracesFromOtlpProtoBytes`: Fixes to handle backwards compatibility changes in proto (#2798)
- `jaeger` receiver: Escape user input used in output (#2815)
- `prometheus` exporter: Ensure same time is used for updated time (#2745)
- `prometheusremotewrite` exporter: Close HTTP response body (#2875)

## v0.23.0 Beta

## 🛑 Breaking changes 🛑

- Move fanout consumers to fanoutconsumer package (#2615)
- Rename ExporterObsReport to Exporter (#2658)
- Rename ProcessorObsReport to Processor (#2657)
- Remove ValidateConfig and add Validate on the Config struct (#2665)
- Rename pdata Size to OtlpProtoSize (#2726)
- Rename [Traces|Metrics|Logs]Consumer to [Traces|Metrics|Logs] (#2761)
- Remove public access for `componenttest.Example*` components:
  - Users of these structs for testing configs should use the newly added `componenttest.Nop*` (update all components name in the config `example*` -> `nop` and use `componenttest.NopComponents()`).
  - Users of these structs for sink like behavior should use `consumertest.*Sink`.

## 💡 Enhancements 💡

- `hostmetrics` receiver: List labels along with respective metrics in metadata (#2662)
- `exporter` helper: Remove obsreport.ExporterContext, always add exporter name as a tag to the metrics (#2682)
- `jaeger` exporter: Change to not use internal data (#2698)
- `kafka` receiver: Change to not use internal data (#2697)
- `zipkin` receiver: Change to not use internal data (#2699)
- `kafka` exporter: Change to not use internal data (#2696)
- Ensure that extensions can be created and started multiple times (#2679)
- Use otlp request in logs wrapper, hide members in the wrapper (#2692)
- Add MetricsWrapper to dissallow access to internal representation (#2693)
- Add TracesWrapper to dissallow access to internal representation (#2721)
- Allow multiple OTLP receivers to be created (#2743)

## 🧰 Bug fixes 🧰

- `prometheus` exporter: Fix to work with standard labels that follow the naming convention of using periods instead of underscores (#2707)
- Propagate name and transport for `prometheus` receiver and exporter (#2680)
- `zipkin` receiver: Ensure shutdown correctness (#2765)

## v0.22.0 Beta

## 🛑 Breaking changes 🛑

- Rename ServiceExtension to just Extension (#2581)
- Remove `consumerdata.TraceData` (#2551)
- Move `consumerdata.MetricsData` to `internaldata.MetricsData` (#2512)
- Remove custom OpenCensus sematic conventions that have equivalent in otel (#2552)
- Move ScrapeErrors and PartialScrapeError to `scrapererror` (#2580)
- Remove support for deprecated unmarshaler `CustomUnmarshaler`, only `Unmarshal` is supported (#2591)
- Remove deprecated componenterror.CombineErrors (#2598)
- Rename `pdata.TimestampUnixNanos` to `pdata.Timestamp` (#2549)

## 💡 Enhancements 💡

- `prometheus` exporter: Re-implement on top of `github.com/prometheus/client_golang/prometheus` and add `metric_expiration` option
- `logging` exporter: Add support for AttributeMap (#2609)
- Add semantic conventions for instrumentation library (#2602)

## 🧰 Bug fixes 🧰

- `otlp` receiver: Fix `Shutdown()` bug (#2564)
- `batch` processor: Fix Shutdown behavior (#2537)
- `logging` exporter: Fix handling the loop for empty attributes (#2610)
- `prometheusremotewrite` exporter: Fix counter name check (#2613)

## v0.21.0 Beta

## 🛑 Breaking changes 🛑

- Remove deprecated function `IsValid` from trace/span ID (#2522)
- Remove accessors for deprecated status code (#2521)

## 💡 Enhancements 💡

- `otlphttp` exporter: Add `compression` option for gzip encoding of outgoing http requests (#2502)
- Add `ScrapeErrors` struct to `consumererror` to simplify errors usage (#2414)
- Add `cors_allowed_headers` option to `confighttp` (#2454)
- Add SASL/SCRAM authentication mechanism on `kafka` receiver and exporter (#2503)

## 🧰 Bug fixes 🧰

- `otlp` receiver: Sets the correct deprecated status code before sending data to the pipeline (#2521)
- Fix `IsPermanent` to account for wrapped errors (#2455)
- `otlp` exporter: Preserve original error messages (#2459)

## v0.20.0 Beta

## 🛑 Breaking changes 🛑

- Rename `samplingprocessor/probabilisticsamplerprocessor` to `probabilisticsamplerprocessor` (#2392)

## 💡 Enhancements 💡

- `hostmetrics` receiver: Refactor to use metrics metadata utilities (#2405, #2406, #2421)
- Add k8s.node semantic conventions (#2425)

## v0.19.0 Beta

## 🛑 Breaking changes 🛑
- Remove deprecated `queued_retry` processor
- Remove deprecated configs from `resource` processor: `type` (set "opencensus.type" key in "attributes.upsert" map instead) and `labels` (use "attributes.upsert" instead).

## 💡 Enhancements 💡

- `hostmetrics` receiver: Refactor load metrics to use generated metrics (#2375)
- Add uptime to the servicez debug page (#2385)
- Add new semantic conventions for AWS (#2365)

## 🧰 Bug fixes 🧰

- `jaeger` exporter: Improve connection state logging (#2239)
- `pdatagen`: Fix slice of values generated code (#2403)
- `filterset` processor: Avoid returning always nil error in strict filterset (#2399)

## v0.18.0 Beta

## 🛑 Breaking changes 🛑
- Rename host metrics according to metrics spec and rename `swap` scraper to `paging` (#2311)

## 💡 Enhancements 💡

- Add check for `NO_WINDOWS_SERVICE` environment variable to force interactive mode on Windows (#2272)
- `hostmetrics` receiver: Add `disk/weighted_io_time` metric (Linux only) (#2312)
- `opencensus` exporter: Add queue-retry (#2307)
- `filter` processor: Filter metrics using resource attributes (#2251)

## 🧰 Bug fixes 🧰

- `fluentforward` receiver: Fix string conversions (#2314)
- Fix zipkinv2 translation error tag handling (#2253)

## v0.17.0 Beta

## 💡 Enhancements 💡

- Default config environment variable expansion (#2231)
- `prometheusremotewrite` exporter: Add batched exports (#2249)
- `memorylimiter` processor: Introduce soft and hard limits (#2250)

## 🧰 Bug fixes 🧰

- Fix nits in pdata usage (#2235)
- Convert status to not be a pointer in the Span (#2242)
- Report the error from `pprof.StartCPUProfile` (#2263)
- Rename `service.Application.SignalTestComplete` to `Shutdown` (#2277)

## v0.16.0 Beta

## 🛑 Breaking changes 🛑

- Rename Push functions to be consistent across signals in `exporterhelper` (#2203)

## 💡 Enhancements 💡

- Change default OTLP/gRPC port number to 4317, also continue receiving on legacy port
  55680 during transition period (#2104).
- `kafka` exporter: Add support for exporting metrics as otlp Protobuf. (#1966)
- Move scraper helpers to its own `scraperhelper` package (#2185)
- Add `componenthelper` package to help build components (#2186)
- Remove usage of custom init/stop in `scraper` and use start/shutdown from `component` (#2193)
- Add more trace annotations, so zpages are more useful to determine failures (#2206)
- Add support to skip TLS verification (#2202)
- Expose non-nullable metric types (#2208)
- Expose non-nullable elements from slices of pointers (#2200)

## 🧰 Bug fixes 🧰

- Change InstrumentationLibrary to be non-nullable (#2196)
- Add support for slices to non-pointers, use non-nullable AnyValue (#2192)
- Fix `--set` flag to work with `{}` in configs (#2162)

## v0.15.0 Beta

## 🛑 Breaking changes 🛑

- Remove legacy metrics, they were marked as legacy for ~12 months #2105

## 💡 Enhancements 💡

- Implement conversion between OpenCensus and OpenTelemetry Summary Metric (#2048)
- Add ability to generate non nullable messages (#2005)
- Implement Summary Metric in Prometheus RemoteWrite Exporter (#2083)
- Add `resource_to_telemetry_conversion` to exporter helper expose exporter settings (#2060)
- Add `CustomRoundTripper` function to httpclientconfig (#2085)
- Allow for more logging options to be passed to `service` (#2132)
- Add config parameters for `jaeger` receiver (#2068)
- Map unset status code for `jaegar` translator as per spec (#2134)
- Add more trace annotations to the queue-retry logic (#2136)
- Add config settings for component telemetry (#2148)
- Use net.SplitHostPort for IPv6 support in `prometheus` receiver (#2154)
- Add --log-format command line option (default to "console") #2177.

## 🧰 Bug fixes 🧰

- `logging` exporter: Add Logging for Summary Datapoint (#2084)
- `hostmetrics` receiver: use correct TCP state labels on Unix systems (#2087)
- Fix otlp_log receiver wrong use of trace measurement (#2117)
- Fix "process/memory/rss" metric units (#2112)
- Fix "process/cpu_seconds" metrics (#2113)
- Add check for nil logger in exporterhelper functions (#2141)
- `prometheus` receiver:
  - Upgrade Prometheus version to fix race condition (#2121)
  - Fix the scraper/discover manager coordination (#2089)
  - Fix panic when adjusting buckets (#2168)

## v0.14.0 Beta

## 🚀 New components 🚀

- `otlphttp` exporter which implements OTLP over HTTP protocol.

## 🛑 Breaking changes 🛑

- Rename consumer.TraceConsumer to consumer.TracesConsumer #1974
- Rename component.TraceReceiver to component.TracesReceiver #1975
- Rename component.TraceProcessor to component.TracesProcessor #1976
- Rename component.TraceExporter to component.TracesExporter #1975
- Deprecate NopExporter, add NopConsumer (#1972)
- Deprecate SinkExporter, add SinkConsumer (#1973)
- Move `tailsampling` processor to contrib (#2012)
- Remove NewAttributeValueSlice (#2028) and mark NewAttributeValue as deprecated (#2022)
- Remove pdata.StringValue (#2021)
- Remove pdata.InitFromAttributeMap, use CopyTo if needed (#2042)
- Remove SetMapVal and SetArrayVal for pdata.AttributeValue (#2039)

## 💡 Enhancements 💡

- `zipkin` exporter: Add queue retry to zipkin (#1971)
- `prometheus` exporter: Add `send_timestamps` option (#1951)
- `filter` processor: Add `expr` pdata.Metric filtering support (#1940, #1996)
- `attribute` processor: Add log support (#1934)
- `logging` exporter: Add index for histogram buckets count (#2009)
- `otlphttp` exporter: Add correct handling of server error responses (#2016)
- `prometheusremotewrite` exporter:
  - Add user agent header to outgoing http request (#2000)
  - Convert histograms to cumulative (#2049)
  - Return permanent errors (#2053)
  - Add external labels (#2044)
- `hostmetrics` receiver: Use scraper controller (#1949)
- Change Span/Trace ID to be byte array (#2001)
- Add `simple` metrics helper to facilitate building pdata.Metrics in receivers (#1540)
- Improve diagnostic logging for exporters (#2020)
- Add obsreport to receiverhelper scrapers (#1961)
- Update OTLP to 0.6.0 and use the new Span Status code (#2031)
- Add support of partial requests for logs and metrics to the exporterhelper (#2059)

## 🧰 Bug fixes 🧰

- `logging` exporter: Added array serialization (#1994)
- `zipkin` receiver: Allow receiver to parse string tags (#1893)
- `batch` processor: Fix shutdown race (#1967)
- Guard for nil data points (#2055)

## v0.13.0 Beta

## 🛑 Breaking changes 🛑

- Host metric `system.disk.time` renamed to `system.disk.operation_time` (#1887)
- Use consumer for sender interface, remove unnecessary receiver address from Runner (#1941)
- Enable sending queue by default in all exporters configured to use it (#1924)
- Removed `groupbytraceprocessor` (#1891)
- Remove ability to configure collection interval per scraper (#1947)

## 💡 Enhancements 💡

- Host Metrics receiver now reports both `system.disk.io_time` and `system.disk.operation_time` (#1887)
- Match spans against the instrumentation library and resource attributes (#928)
- Add `receiverhelper` for creating flexible "scraper" metrics receiver (#1886, #1890, #1945, #1946)
- Migrate `tailsampling` processor to new OTLP-based internal data model and add Composite Sampler (#1894)
- Metadata Generator: Change Metrics fields to implement an interface with new methods (#1912)
- Add unmarshalling for `pdata.Traces` (#1948)
- Add debug-level message on error for `jaeger` exporter (#1964)

## 🧰 Bug fixes 🧰

- Fix bug where the service does not correctly start/stop the log exporters (#1943)
- Fix Queued Retry Unusable without Batch Processor (#1813) - (#1930)
- `prometheus` receiver: Log error message when `process_start_time_seconds` gauge is missing (#1921)
- Fix trace jaeger conversion to internal traces zero time bug (#1957)
- Fix panic in otlp traces to zipkin (#1963)
- Fix OTLP/HTTP receiver's path to be /v1/traces (#1979)

## v0.12.0 Beta

## 🚀 New components 🚀

- `configauth` package with the auth settings that can be used by receivers (#1807, #1808, #1809, #1810)
- `perfcounters` package that uses perflib for host metrics receiver (#1835, #1836, #1868, #1869, #1870)

## 💡 Enhancements 💡

- Remove `queued_retry` and enable `otlp` metrics receiver in default config (#1823, #1838)
- Add `limit_percentage` and `spike_limit_percentage` options to `memorylimiter` processor (#1622)
- `hostmetrics` receiver:
  - Collect additional labels from partitions in the filesystems scraper (#1858)
  - Add filters for mount point and filesystem type (#1866)
- Add cloud.provider semantic conventions (#1865)
- `attribute` processor: Add log support (#1783)
- Deprecate OpenCensus-based internal data structures (#1843)
- Introduce SpanID data type, not yet used in Protobuf messages ($1854, #1855)
- Enable `otlp` trace by default in the released docker image (#1883)
- `tailsampling` processor: Combine batches of spans into a single batch (#1864)
- `filter` processor: Update to use pdata (#1885)
- Allow MSI upgrades (#1914)

## 🧰 Bug fixes 🧰

- `prometheus` receiver: Print a more informative message about 'up' metric value (#1826)
- Use custom data type and custom JSON serialization for traceid (#1840)
- Skip creation of redundant nil resource in translation from OC if there are no combined metrics (#1803)
- `tailsampling` processor: Only send to next consumer once (#1735)
- Report Windows pagefile usage in bytes (#1837)
- Fix issue where Prometheus SD config cannot be parsed (#1877)

## v0.11.0 Beta

## 🛑 Breaking changes 🛑

- Rename service.Start() to Run() since it's a blocking call
- Fix slice Append to accept by value the element in pdata
- Change CreateTraceProcessor and CreateMetricsProcessor to use the same parameter order as receivers/logs processor and exporters.
- Prevent accidental use of LogsToOtlp and LogsFromOtlp and the OTLP data structs (#1703)
- Remove SetType from configmodels, ensure all registered factories set the type in config (#1798)
- Move process telemetry to service/internal (#1794)

## 💡 Enhancements 💡

- Add map and array attribute value type support (#1656)
- Add authentication support to kafka (#1632)
- Implement InstrumentationLibrary translation to jaeger (#1645)
- Add public functions to export pdata to ExportXServicesRequest Protobuf bytes (#1741)
- Expose telemetry level in the configtelemetry (#1796)
- Add configauth package (#1807)
- Add config to docker image (#1792)

## 🧰 Bug fixes 🧰

- Use zap int argument for int values instead of conversion (#1779)
- Add support for gzip encoded payload in OTLP/HTTP receiver (#1581)
- Return proto status for OTLP receiver when failed (#1788)

## v0.10.0 Beta

## 🛑 Breaking changes 🛑

- **Update OTLP to v0.5.0, incompatible metrics protocol.**
- Remove support for propagating summary metrics in OtelCollector.
  - This is a temporary change, and will affect mostly OpenCensus users who use metrics.

## 💡 Enhancements 💡
- Support zipkin proto in `kafka` receiver (#1646)
- Prometheus Remote Write Exporter supporting Cortex (#1577, #1643)
- Add deployment environment semantic convention (#1722)
- Add logs support to `batch` and `resource` processors (#1723, #1729)

## 🧰 Bug fixes 🧰
- Identify config error when expected map is other value type (#1641)
- Fix Kafka receiver closing ready channel multiple times (#1696)
- Fix a panic issue while processing Zipkin spans with an empty service name (#1742)
- Zipkin Receiver: Always set the endtime (#1750)

## v0.9.0 Beta

## 🛑 Breaking changes 🛑

- **Remove old base factories**:
  - `ReceiverFactoryBase` (#1583)
  - `ProcessorFactoryBase` (#1596)
  - `ExporterFactoryBase` (#1630)
- Remove logs factories and merge with normal factories (#1569)
- Remove `reconnection_delay` from OpenCensus exporter (#1516)
- Remove `ConsumerOld` interfaces (#1631)

## 🚀 New components 🚀
- `prometheusremotewrite` exporter: Send metrics data in Prometheus TimeSeries format to Cortex or any Prometheus (#1544)
- `kafka` receiver: Receive traces from Kafka (#1410)

## 💡 Enhancements 💡
- `kafka` exporter: Enable queueing, retry, timeout (#1455)
- Add `Headers` field in HTTPClientSettings (#1552)
- Change OpenCensus receiver (#1556) and exporter (#1571) to the new interfaces
- Add semantic attribute for `telemetry.auto.version` (#1578)
- Add uptime and RSS memory self-observability metrics (#1549)
- Support conversion for OpenCensus `SameProcessAsParentSpan` (#1629)
- Access application version in components (#1559)
- Make Kafka payload encoding configurable (#1584)

## 🧰 Bug fixes 🧰
- Stop further processing if `filterprocessor` filters all data (#1500)
- `processscraper`: Use same scrape time for all data points coming from same process (#1539)
- Ensure that time conversion for 0 returns nil timestamps or Time where IsZero returns true (#1550)
- Fix multiple exporters panic (#1563)
- Allow `attribute` processor for external use (#1574)
- Do not duplicate filesystem metrics for devices with many mount points (#1617)

## v0.8.0 Beta

## 🚀 New components 🚀

- `groupbytrace` processor that waits for a trace to be completed (#1362)

## 💡 Enhancements 💡

- Migrate `zipkin` receiver/exporter to the new interfaces (#1484)
- Migrate `prometheus` receiver/exporter to the new interfaces (#1477, #1515)
- Add new FactoryUnmarshaler support to all components, deprecate old way (#1468)
- Update `fileexporter` to write data in OTLP (#1488)
- Add extension factory helper (#1485)
- Host scrapers: Use same scrape time for all data points coming from same source (#1473)
- Make logs SeverityNumber publicly available (#1496)
- Add recently included conventions for k8s and container resources (#1519)
- Add new config StartTimeMetricRegex to `prometheus` receiver (#1511)
- Convert Zipkin receiver and exporter to use OTLP (#1446)

## 🧰 Bug fixes 🧰

- Infer OpenCensus resource type based on OpenTelemetry's semantic conventions (#1462)
- Fix log adapter in `prometheus` receiver (#1493)
- Avoid frequent errors for process telemetry on Windows (#1487)

## v0.7.0 Beta

## 🚀 New components 🚀

- Receivers
  - `fluentfoward` runs a TCP server that accepts events via the [Fluent Forward protocol](https://github.com/fluent/fluentd/wiki/Forward-Protocol-Specification-v1) (#1173)
- Exporters
  - `kafka` exports traces to Kafka (#1439)
- Extensions
  - **Experimental** `fluentbit` facilitates running a FluentBit subprocess of the collector (#1381)

## 💡 Enhancements 💡

- Updated `golang/protobuf` from v1.3.5 to v1.4.2 (#1308)
- Updated `opencensus-proto` from v0.2.1 to v0.3.0 (#1308)
- Added round_robin `balancer_name` as an option to gRPC client settings (#1353)
- `hostmetrics` receiver
  - Switch to using perf counters to get disk io metrics on Windows (#1340)
  - Add device filter for file system (#1379) and disk (#1378) scrapers
  - Record process physical & virtual memory stats separately (#1403)
  - Scrape system.disk.time on Windows (#1408)
  - Add disk.pending_operations metric (#1428)
  - Add network interface label to network metrics (#1377)
- Add `exporterhelper` (#1351) and `processorhelper` (#1359) factories
- Update OTLP to latest version (#1384)
- Disable timeout, retry on failure and sending queue for `logging` exporter (#1400)
- Add support for retry and sending queue for `jaeger` exporter (#1401)
- Add batch size bytes metric to `batch` processor (#1270)
- `otlp` receiver: Add Log Support (#1444)
- Allow to configure read/write buffer sizes for http Client (#1447)
- Update DB conventions to latest and add exception conventions (#1452)

## 🧰 Bug fixes 🧰

- Fix `resource` processor for old metrics (#1412)
- `jaeger` receiver: Do not try to stop if failed to start. Collector service will do that (#1434)

## v0.6.0 Beta

## 🛑 Breaking changes 🛑

- Renamed the metrics generated by `hostmetrics` receiver to match the (currently still pending) OpenTelemetry system metric conventions (#1261) (#1269)
- Removed `vmmetrics` receiver (#1282)
- Removed `cpu` scraper `report_per_cpu` config option (#1326)

## 💡 Enhancements 💡

- Added disk merged (#1267) and process count (#1268) metrics to `hostmetrics`
- Log metric data points in `logging` exporter (#1258)
- Changed the `batch` processor to not ignore the errors returned by the exporters (#1259)
- Build and publish MSI (#1153) and DEB/RPM packages (#1278, #1335)
- Added batch size metric to `batch` processor (#1241)
- Added log support for `memorylimiter` processor (#1291) and `logging` exporter (#1298)
- Always add tags for `observability`, other metrics may use them (#1312)
- Added metrics support (#1313) and allow partial retries in `queued_retry` processor (#1297)
- Update `resource` processor: introduce `attributes` config parameter to specify actions on attributes similar to `attributes` processor, old config interface is deprecated (#1315)
- Update memory state labels for non-Linux OSs (#1325)
- Ensure tcp connection value is provided for all states, even when count is 0 (#1329)
- Set `batch` processor channel size to num cpus (#1330)
- Add `send_batch_max_size` config parameter to `batch` processor enforcing hard limit on batch size (#1310)
- Add support for including a per-RPC authentication to gRPC settings (#1250)

## 🧰 Bug fixes 🧰

- Fixed OTLP waitForReady, not set from config (#1254)
- Fixed all translation diffs between OTLP and Jaeger (#1222)
- Disabled `process` scraper for any non Linux/Windows OS (#1328)

## v0.5.0 Beta

## 🛑 Breaking changes 🛑

- **Update OTLP to v0.4.0 (#1142)**: Collector will be incompatible with any other sender or receiver of OTLP protocol
of different versions
- Make "--new-metrics" command line flag the default (#1148)
- Change `endpoint` to `url` in Zipkin exporter config (#1186)
- Change `tls_credentials` to `tls_settings` in Jaegar receiver config (#1233)
- OTLP receiver config change for `protocols` to support mTLS (#1223)
- Remove `export_resource_labels` flag from Zipkin exporter (#1163)

## 🚀 New components 🚀

- Receivers
  - Added process scraper to the `hostmetrics` receiver (#1047)

## 💡 Enhancements 💡

- otlpexporter: send configured headers in request (#1130)
- Enable Collector to be run as a Windows service (#1120)
- Add config for HttpServer (#1196)
- Allow cors in HTTPServerSettings (#1211)
- Add a generic grpc server settings config, cleanup client config (#1183)
- Rely on gRPC to batch and loadbalance between connections instead of custom logic (#1212)
- Allow to tune the read/write buffers for gRPC clients (#1213)
- Allow to tune the read/write buffers for gRPC server (#1218)

## 🧰 Bug fixes 🧰

- Handle overlapping metrics from different jobs in prometheus exporter (#1096)
- Fix handling of SpanKind INTERNAL in OTLP OC translation (#1143)
- Unify zipkin v1 and v2 annotation/tag parsing logic (#1002)
- mTLS: Add support to configure client CA and enforce ClientAuth (#1185)
- Fixed untyped Prometheus receiver bug (#1194)
- Do not embed ProtocolServerSettings in gRPC (#1210)
- Add Context to the missing CreateMetricsReceiver method (#1216)

## v0.4.0 Beta

Released 2020-06-16

## 🛑 Breaking changes 🛑

- `isEnabled` configuration option removed (#909) 
- `thrift_tchannel` protocol moved from `jaeger` receiver to `jaeger_legacy` in contrib (#636) 

## ⚠️ Major changes ⚠️

- Switch from `localhost` to `0.0.0.0` by default for all receivers (#1006)
- Internal API Changes (only impacts contributors)
  - Add context to `Start` and `Stop` methods in the component (#790)
  - Rename `AttributeValue` and `AttributeMap` method names (#781)
(other breaking changes in the internal trace data types)
  - Change entire repo to use the new vanityurl go.opentelemetry.io/collector (#977)

## 🚀 New components 🚀

- Receivers
  - `hostmetrics` receiver with CPU (#862), disk (#921), load (#974), filesystem (#926), memory (#911), network (#930), and virtual memory (#989) support
- Processors
  - `batch` for batching received metrics (#1060) 
  - `filter` for filtering (dropping) received metrics (#1001) 

## 💡 Enhancements 💡

- `otlp` receiver implement HTTP X-Protobuf (#1021)
- Exporters: Support mTLS in gRPC exporters (#927) 
- Extensions: Add `zpages` for service (servicez, pipelinez, extensions) (#894) 

## 🧰 Bug fixes 🧰

- Add missing logging for metrics at `debug` level (#1108) 
- Fix setting internal status code in `jaeger` receivers (#1105) 
- `zipkin` export fails on span without timestamp when used with `queued_retry` (#1068) 
- Fix `zipkin` receiver status code conversion (#996) 
- Remove extra send/receive annotations with using `zipkin` v1 (#960)
- Fix resource attribute mutation bug when exporting in `jaeger` proto (#907) 
- Fix metric/spans count, add tests for nil entries in the slices (#787) 


## 🧩 Components 🧩

### Traces

| Receivers | Processors | Exporters |
|:----------:|:-----------:|:----------:|
| Jaeger | Attributes | File |
| OpenCensus | Batch | Jaeger |
| OTLP | Memory Limiter | Logging |
| Zipkin | Queued Retry | OpenCensus |
| | Resource | OTLP |
| | Sampling | Zipkin |
| | Span ||

### Metrics

| Receivers | Processors | Exporters |
|:----------:|:-----------:|:----------:|
| HostMetrics | Batch | File |
| OpenCensus | Filter | Logging |
| OTLP | Memory Limiter | OpenCensus |
| Prometheus || OTLP |
| VM Metrics || Prometheus |

### Extensions

- Health Check
- Performance Profiler
- zPages


## v0.3.0 Beta

Released 2020-03-30

### Breaking changes

-  Make prometheus receiver config loading strict. #697 
Prometheus receiver will now fail fast if the config contains unused keys in it.

### Changes and fixes

- Enable best effort serve by default of Prometheus Exporter (https://github.com/orijtech/prometheus-go-metrics-exporter/pull/6)
- Fix null pointer exception in the logging exporter #743 
- Remove unnecessary condition to have at least one processor #744 

### Components

| Receivers / Exporters | Processors | Extensions |
|:---------------------:|:-----------:|:-----------:|
| Jaeger | Attributes | Health Check |
| OpenCensus | Batch | Performance Profiler |
| OpenTelemetry | Memory Limiter | zPages |
| Zipkin | Queued Retry | |
| | Resource | |
| | Sampling | |
| | Span | |


## v0.2.8 Alpha

Alpha v0.2.8 of OpenTelemetry Collector

- Implemented OTLP receiver and exporter.
- Added ability to pass config to the service programmatically (useful for custom builds).
- Improved own metrics / observability.
- Refactored component and factory interface definitions (breaking change #683) 


## v0.2.7 Alpha

Alpha v0.2.7 of OpenTelemetry Collector

- Improved error handling on shutdown
- Partial implementation of new metrics (new obsreport package)
- Include resource labels for Zipkin exporter
- New `HASH` action to attribute processor



## v0.2.6 Alpha

Alpha v0.2.6 of OpenTelemetry Collector.
- Update metrics prefix to `otelcol` and expose command line argument to modify the prefix value.
- Extend Span processor to have include/exclude span logic.
- Batch dropped span now emits zero when no spans are dropped.


## v0.2.5 Alpha

Alpha v0.2.5 of OpenTelemetry Collector.

- Regexp-based filtering of spans based on service names.
- Ability to choose strict or regexp matching for include/exclude filters.


## v0.2.4 Alpha

Alpha v0.2.4 of OpenTelemetry Collector.

- Regexp-based filtering of span names.
- Ability to extract attributes from span names and rename span.
- File exporter for debugging.
- Span processor is now enabled by default.


## v0.2.3 Alpha

Alpha v0.2.3 of OpenTelemetry Collector.

Changes:
21a70d6 Add a memory limiter processor (#498)
9778b16 Refactor Jaeger Receiver config (#490)
ec4ad0c Remove workers from OpenCensus receiver implementation (#497)
4e01fa3 Update k8s config to use opentelemetry docker image and configuration (#459)


## v0.2.2 Alpha

Alpha v0.2.2 of OpenTelemetry Collector.

Main changes visible to users since previous release:

- Improved Testbed and added more E2E tests.
- Made component interfaces more uniform (this is a breaking change).

Note: v0.2.1 never existed and is skipped since it was tainted in some dependencies.


## v0.2.0 Alpha

Alpha v0.2 of OpenTelemetry Collector.

Docker image: omnition/opentelemetry-collector:v0.2.0 (we are working on getting this under an OpenTelemetry org)

Main changes visible to users since previous release:

* Rename from `service` to `collector`, the binary is now named `otelcol`

* Configuration reorganized and using strict mode

* Concurrency issues for pipelines transforming data addressed

Commits:

```terminal
0e505d5 Refactor config: pipelines now under service (#376)
402b80c Add Capabilities to Processor and use for Fanout cloning decision (#374)
b27d824 Use strict mode to read config (#375)
d769eb5 Fix concurrency handling when data is fanned out (#367)
dc6b290 Rename all github paths from opentelemtry-service to opentelemetry-collector (#371)
d038801 Rename otelsvc to otelcol (#365)
c264e0e Add Include/Exclude logic for Attributes Processor (#363)
8ce427a Pin a commit for Prometheus dependency in go.mod (#364)
2393774 Bump Jaeger version to 1.14.0 (latest) (#349)
63362d5 Update testbed modules (#360)
c0e2a27 Change dashes to underscores to separate words in config files (#357)
7609eaa Rename OpenTelemetry Service to Collector in docs and comments (#354)
bc5b299 Add common gRPC configuration settings (#340)
b38505c Remove network access popups on macos (#348)
f7727d1 Fixed loop variable pointer bug in jaeger translator (#341)
958beed Ensure that ConsumeMetricsData() is not passed empty metrics in the Prometheus receiver (#345)
0be295f Change log statement in Prometheus receiver from info to debug. (#344)
d205393 Add Owais to codeowners (#339)
8fa6afe Translate OC resource labels to Jaeger process tags (#325)
```


## v0.0.2 Alpha

Alpha release of OpenTelemetry Service.

Docker image: omnition/opentelemetry-service:v0.0.2 (we are working on getting this under an OpenTelemetry org)

Main changes visible to users since previous release:

```terminal
8fa6afe Translate OC resource labels to Jaeger process tags (#325)
047b0f3 Allow environment variables in config (#334)
96c24a3 Add exclude/include spans option to attributes processor (#311)
4db0414 Allow metric processors to be specified in pipelines (#332)
c277569 Add observability instrumentation for Prometheus receiver (#327)
f47aa79 Add common configuration for receiver tls (#288)
a493765 Refactor extensions to new config format (#310)
41a7afa Add Span Processor logic
97a71b3 Use full name for the metrics and spans created for observability (#316)
fed4ed2 Add support to record metrics for metricsexporter (#315)
5edca32 Add include_filter configuration to prometheus receiver (#298)
0068d0a Passthrough CORS allowed origins (#260)
```


## v0.0.1 Alpha

This is the first alpha release of OpenTelemetry Service.

Docker image: omnition/opentelemetry-service:v0.0.1


[v0.3.0]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.2.10...v0.3.0
[v0.2.10]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.2.8...v0.2.10
[v0.2.8]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.2.7...v0.2.8
[v0.2.7]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.2.6...v0.2.7
[v0.2.6]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.2.5...v0.2.6
[v0.2.5]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.2.4...v0.2.5
[v0.2.4]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.2.3...v0.2.4
[v0.2.3]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.2.2...v0.2.3
[v0.2.2]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.2.0...v0.2.2
[v0.2.0]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.0.2...v0.2.0
[v0.0.2]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.0.1...v0.0.2
[v0.0.1]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.0.1
