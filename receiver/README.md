A variety of receivers are available to the OpenCensus Service (both Agent and Collector)

__Currently there are some inconsistencies between Agent and Collector configuration, those will be addressed by issue
[#135](https://github.com/census-instrumentation/opencensus-service/issues/135).__ 

## OpenCensus

This receiver receives spans from OpenCensus instrumented applications and translates them into the internal span types that are then sent to the collector/exporters.

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

### Collector Differences
(To be fixed via [#135](https://github.com/census-instrumentation/opencensus-service/issues/135))

By default this receiver is ALWAYS started on the OpenCensus Collector, it can be disabled via command-line by
using `--receive-oc-trace=false`. On the Collector only the port can be configured, example:

```yaml
receivers:
  opencensus:
    port: 55678
```

## Jaeger

This receiver receives spans from Jaeger collector HTTP and Thrift uploads and translates them into the internal span types that are then sent to the collector/exporters.

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

## Zipkin

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
