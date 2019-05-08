A variety of exporters are available to the OpenCensus Service (both Agent and Collector)

## Collector

### Intelligent Sampling

The Collector features intelligent (tail-based) sampling. In addition to different configuration
options it features a variety of different policies.
```yaml
sampling:
  mode: tail
  # amount of time from seeing the first span in a trace until making the sampling decision
  decision-wait: 10s
  # maximum number of traces kept in the memory
  num-traces: 10000
  policies:
    # user-defined policy name
    my-rate-limiting:
      # exporters the policy applies to
      exporters:
        - jaeger
        - omnition
      policy: rate-limiting
      configuration:
        spans-per-second: 1000
    my-string-attribute-filter:
      exporters:
        - jaeger
        - omnition
      policy: string-attribute-filter
      configuration:
        key: key1
        values:
          - value1
          - value2
    my-numeric-attribute-filter:
      exporters:
        - jaeger
        - omnition
      policy: numeric-attribute-filter
      configuration:
        key: key1
        min-value: 0
        max-value: 100
    my-always-sample:
      exporters:
        - jaeger
        - omnition
      policy: always-sample
```

### Queued Exporters

In addition to the normal `exporters`, the OpenCensus Collector supports a special configuration.
`queued-exporters` offer bounded buffer retry logic for multiple destinations.

```yaml
queued-exporters:
  omnition: # A friendly name for the processor
    batching:
      enable: false
      # sets the time, in seconds, after which a batch will be sent regardless of size
      timeout: 1
      # number of spans which after hit, will trigger it to be sent
      send-batch-size: 8192
    # num-workers is the number of queue workers that will be dequeuing batches and sending them out (default is 10)
    num-workers: 2
    # queue-size is the maximum number of batches allowed in the queue at a given time (default is 5000)
    queue-size: 100
    # retry-on-failure indicates whether queue processor should retry span batches in case of processing failure (default is true)
    retry-on-failure: true
    # backoff-delay is the amount of time a worker waits after a failed send before retrying (default is 5 seconds)
    backoff-delay: 3s
    # sender-type is the type of sender used by this processor, the default is an invalid sender so it forces one to be specified
    sender-type: jaeger-thrift-http
    # configuration of the selected sender-type, in this example jaeger-thrift-http. Which supports 3 settings:
    # collector-endpoint: address of Jaeger collector thrift-http endpoint
    # headers: a map of any additional headers to be sent with each batch (e.g.: api keys, etc)
    # timeout: the timeout for the sender to consider the operation as failed
    jaeger-thrift-http:
      collector-endpoint: "https://ingest.omnition.io"
      headers: { "x-omnition-api-key": "00000000-0000-0000-0000-000000000001" }
      timeout: 5s
    # Non-sender exporters can now also be used by setting the exporters section in queued-exporters.
    exporters:
      opencensus:
        endpoint: "127.0.0.1:55566"
        compression: "gzip"
  my-org-jaeger: # A second processor with its own configuration options
    num-workers: 2
    queue-size: 100
    retry-on-failure: true
    backoff-delay: 3s
    sender-type: jaeger-thrift-http
    jaeger-thrift-http:
      collector-endpoint: "http://jaeger.local:14268/api/traces"
      timeout: 5s
```
