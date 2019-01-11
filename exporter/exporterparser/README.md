A variety of exporters are available to the OpenCensus Service (both Agent and Collector)

## Collector

In addition to the normal `exporters`, the OpenCensus Collector supports a special configuration.
`queued-exporters` offer bounded buffer retry logic for multiple destinations.

```yaml
queued-exporters:
  omnition: # A friendly name for the processor
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
