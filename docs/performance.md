# OpenTelemetry Collector Performance

The performance numbers that follow were generated using version 0.1.3 of the
OpenTelemetry Collector, are applicable primarily to the OpenTelemetry Collector and
are measured only for traces. In the future, more configurations will be tested.

Note with the OpenTelemetry Agent you can expect as good if not better performance
with lower resource utilization. This is because the OpenTelemetry Agent does not
today support features such as batching or retries and will not support
tail_sampling.

It is important to note that the performance of the OpenTelemetry Collector depends
on a variety of factors including:

* The receiving format: OpenTelemetry (55678), Jaeger thrift (14268) or Zipkin v2 JSON (9411)
* The size of the spans (tests are based on number of attributes): 20
* Whether tail_sampling is enabled or not
* CPU / Memory allocation
* Operating System: Linux

## Testing

Testing was completed on Linux using the [Synthetic Load Generator
utility](https://github.com/Omnition/synthetic-load-generator) running for a
minimum of one hour (i.e. sustained rate). You can be reproduce these results in
your own environment using the parameters described in this document. It is
important to note that this utility has a few configurable parameters which can
impact the results of the tests. The parameters used are defined below.

* FlushInterval(ms) [default: 1000]
* MaxQueueSize [default: 100]
* SubmissionRate(spans/sec): 100,000

## Results without tail-based sampling

| Span<br>Format    | CPU<br>(2+ GHz) | RAM<br>(GB) | Sustained<br>Rate | Recommended<br>Maximum |
| :---:             | :---:           | :---:       | :---:             | :---:                  |
| OpenTelemetry        | 1               | 2           | ~12K              | 10K                    |
| OpenTelemetry        | 2               | 4           | ~24K              | 20K                    |
| Jaeger Thrift     | 1               | 2           | ~14K              | 12K                    |
| Jaeger Thrift     | 2               | 4           | ~27.5K            | 24K                    |
| Zipkin v2 JSON    | 1               | 2           | ~10.5K            | 9K                     |
| Zipkin v2 JSON    | 2               | 4           | ~22K              | 18K                    |

If you are NOT using tail-based sampling and you need higher rates then you can
either:

* Divide traffic to different collector (e.g. by region)
* Scale-up by adding more resources (CPU/RAM)
* Scale-out by putting one or more collectors behind a load balancer or k8s
service

## Results with tail-based sampling

> Note: Additional memory is required for tail-based sampling

| Span<br>Format    | CPU<br>(2+ GHz) | RAM<br>(GB) | Sustained<br>Rate | Recommended<br>Maximum |
| :---:             | :---:           | :---:       | :---:             | :---:                  |
| OpenTelemetry        | 1               | 2           | ~9K               | 8K                     |
| OpenTelemetry        | 2               | 4           | ~18K              | 16K                    |
| Jaeger Thrift     | 1               | 6           | ~11.5K            | 10K                    |
| Jaeger Thrift     | 2               | 8           | ~23K              | 20K                    |
| Zipkin v2 JSON    | 1               | 6           | ~8.5K             | 7K                     |
| Zipkin v2 JSON    | 2               | 8           | ~16K              | 14K                    |

If you are using tail-based sampling and you need higher rates then you can
either:

* Scale-up by adding more resources (CPU/RAM)
* Scale-out by putting one or more collectors behind a load balancer or k8s
service, but the load balancer must support traceID-based routing (i.e. all
spans for a given traceID need to be received by the same collector instance)
