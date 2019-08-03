**Note** This documentation is still in progress. For any questions, please
reach out in the [OpenTelemetry Gitter](https://gitter.im/open-telemetry/opentelemetry-service)
or refer to the [issues page](https://github.com/open-telemetry/opentelemetry-service/issues).

# Receivers
A receiver is how data gets into OpenTelemetry Service. Generally, a receiver
accepts data in a specified format and can support traces and/or metrics. The
format of the traces and metrics supported are receiver specific.

Supported receivers (sorted alphabetically):
- [Jaeger Receiver](#jaeger)
- [OpenCensus Receiver](#opencensus)
- [Prometheus Receiver](#prometheus)
- [VM Metrics Receiver](#vmmetrics)
- [Zipkin Receiver](#zipkin)

## Configuring Receiver(s)
TODO - Add what a fullname is and how that is referenced in other parts of the
configuration. Describe the common receiver settings: endpoint, disabled, etc.

## <a name="opencensus"></a>OpenCensus Receiver
**Traces and metrics are supported.**

This receiver receives spans from [OpenCensus](https://opencensus.io/) instrumented applications and translates them into the internal format sent to processors and exporters in the pipeline.

Its address can be configured in the YAML configuration file under section "receivers", subsection "opencensus" and field "address". The syntax of the field "address" is `[address|host]:<port-number>`.

For example:

```yaml
receivers:
  opencensus:
    address: "127.0.0.1:55678"
```
### Writing with HTTP/JSON 

The OpenCensus receiver for the agent can receive trace export calls via
HTTP/JSON in addition to gRPC. The HTTP/JSON address is the same as gRPC as the
protocol is recognized and processed accordingly.

To write traces with HTTP/JSON, `POST` to `[address]/v1/trace`. The JSON message
format parallels the gRPC protobuf format, see this [OpenApi spec for it](https://github.com/census-instrumentation/opencensus-proto/blob/master/gen-openapi/opencensus/proto/agent/trace/v1/trace_service.swagger.json).

The HTTP/JSON endpoint can also optionally 
[CORS](https://fetch.spec.whatwg.org/#cors-protocol), which is enabled by
specifying a list of allowed CORS origins in the `cors_allowed_origins` field:

```yaml
receivers:
  opencensus:
    address: "localhost:55678"
    cors_allowed_origins:
    - http://test.com
    # Origins can have wildcards with *, use * by itself to match any origin.
    - https://*.example.com  
```

### Deprecated YAML Configurations
**Note**: This isn't a full list of deprecated OpenCensus YAML configurations. If something is missing, please expand the documentation
or open an issue.

### Collector Differences
TODO(ccaraman) - Delete all references to opencensus-service issue 135 in follow up pr.
(To be fixed via [#135](https://github.com/census-instrumentation/opencensus-service/issues/135))

By default this receiver is ALWAYS started on the OpenCensus Collector, it can be disabled via command-line by
using `--receive-oc-trace=false`. On the Collector only the port can be configured, example:

```yaml
receivers:
  opencensus:
    port: 55678

    # Settings below are only available on collector.

    # Changes the maximum msg size that can be received (default is 4MiB).
    # See https://godoc.org/google.golang.org/grpc#MaxRecvMsgSize for more information.
    max-recv-msg-size-mib: 32
    
    # Limits the maximum number of concurrent streams for each receiver transport (default is 100).
    # See https://godoc.org/google.golang.org/grpc#MaxConcurrentStreams for more information.
    max-concurrent-streams: 20

    # Controls the keepalive settings, typically used to help scenarios in which the senders have 
    # load-balancers or proxies between them and the collectors.
    keepalive:

      # This section controls the https://godoc.org/google.golang.org/grpc/keepalive#ServerParameters.
      # These are typically used to help load balancers by periodically terminating connections, or keeping
      # connections alive (preventing RSTs by proxies) when needed for bursts of data following periods of
      # inactivity.
      server-parameters:
        # max-connection-idle is the amount of time after which an idle connection would be closed,
        # the default is infinity.
        max-connection-idle: 90s
        # max-connection-age is the maximum amount of time a connection may exist before it is closed,
        # the default is infinity.
        max-connection-age: 180s
        # max-connection-age-grace is an additive period after max-connection-age for which the connection
        # will be forcibly closed. The default is infinity.
        max-connection-age-grace: 10s
        # time is a duration for which, if the server doesn't see any activity it pings the client to see
        # if the transport is still alive. The default is 2 hours.
        time: 30s
        # timeout is the wait time after a ping that the server waits for the response before closing the
        # connection. The default is 20 seconds.
        timeout: 5s 

      # This section controls the https://godoc.org/google.golang.org/grpc/keepalive#EnforcementPolicy.
      # It is used to set keepalive enforcement policy on the server-side. Server will close connection
      # with a client that violates this policy. 
      enforcement-policy:
        # min-time is the minimum amount of time a client should wait before sending a keepalive ping.
        # The default value is 5 minutes.
        min-time: 10s
        # permit-without-stream if true, server allows keepalive pings even when there are no active
        # streams(RPCs). The default is false.
        permit-without-stream: true
```


## <a name="jaeger"></a>Jaeger Receiver
**Only traces are supported.**

This receiver receives spans from Jaeger collector HTTP and Thrift uploads and translates them into the internal span types that are then sent to the collector/exporters.
Only traces are supported. This receiver does not support metrics.

Its address can be configured in the YAML configuration file under section "receivers", subsection "jaeger" and fields "collector_http_port", "collector_thrift_port".

For example:

```yaml
receivers:
  jaeger:
    collector_thrift_port: 14267
    collector_http_port: 14268
```

### Collector Differences
(To be fixed via [#135](https://github.com/census-instrumentation/opencensus-service/issues/135))
 
On the Collector Jaeger reception at the default ports can be enabled via command-line `--receive-jaeger`, and the name of the fields is slightly different:

```yaml
receivers:
  jaeger:
    jaeger-thrift-tchannel-port: 14267
    jaeger-thrift-http-port: 14268
```

## <a name="prometheus"></a>Prometheus Receiver
**Only metrics are supported.**

This receiver is a drop-in replacement for getting Prometheus to scrape your services. Just like you would write in a
YAML configuration file before starting Prometheus, such as with:
```shell
prometheus --config.file=prom.yaml
```

you can copy and paste that same configuration under section
```yaml
receivers:
  prometheus:
    config:
```

such as:
```yaml
receivers:
    prometheus:
      config:
        scrape_configs:
          - job_name: 'opencensus_service'
            scrape_interval: 5s
            static_configs:
              - targets: ['localhost:8889']

          - job_name: 'jdbc_apps'
            scrape_interval: 3s
            static_configs:
              - targets: ['localhost:9777']
```

## <a name="vmmetrics"></a>VM Metrics Receiver
**Only metrics are supported.**

<Add more information - I'm lonely.>

## <a name="zipkin"></a>Zipkin Receiver
**Only traces are supported.**

This receiver receives spans from Zipkin (V1 and V2) HTTP uploads and translates them into the internal span types that are then sent to the collector/exporters.

Its address can be configured in the YAML configuration file under section "receivers", subsection "zipkin" and field "address".  The syntax of the field "address" is `[address|host]:<port-number>`.

For example:

```yaml
receivers:
  zipkin:
    address: "127.0.0.1:9411"
```

### Collector Differences
(To be fixed via [#135](https://github.com/census-instrumentation/opencensus-service/issues/135))

On the Collector Zipkin reception at the port 9411 can be enabled via command-line `--receive-zipkin`. On the Collector only the port can be configured, example:

```yaml
receivers:
  zipkin:
    port: 9411
```

## Common Configuration Errors
<Fill this in as we go with common gotchas experienced by users. These should eventually be made apart of the validation test suite.>
