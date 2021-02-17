# Monitoring

Many metrics are provided by the Collector for its monitoring. Below some
key recommendations for alerting and monitoring are listed. All metrics
referenced below are using the `--new-metrics` option that is enabled by
default.

## Critical Monitoring

### Data Loss

Use rate of `otelcol_processor_dropped_spans > 0` and
`otelcol_processor_dropped_metric_points > 0` to detect data loss, depending on
the requirements set up a minimal time window before alerting, avoiding
notifications for small losses that are not considered outages or within the
desired reliability level.

### Low on CPU Resources

This depends on the CPU metrics available on the deployment, eg.:
`kube_pod_container_resource_limits_cpu_cores` for Kubernetes. Let's call it
`available_cores` below. The idea here is to have an upper bound of the number
of available cores, and the maximum expected ingestion rate considered safe,
let's call it `safe_rate`, per core. This should trigger increase of resources/
instances (or raise an alert as appropriate) whenever 
`(actual_rate/available_cores) < safe_rate`.

The `safe_rate` depends on the specific configuration being used.
// TODO: Provide reference `safe_rate` for a few selected configurations.

## Secondary Monitoring

### Queue Length

Most exporters offer a [queue/retry mechanism](../exporter/exporterhelper/README.md)
that is recommended as the retry mechanism for the Collector and as such should
be used in any production deployment.

**TODO:** Add metric to monitor queue length.

Currently, the queue/retry mechanism only supports logging for monitoring. Check
the logs for messages like `"Dropping data because sending_queue is full"`.

### Receive Failures

Sustained rates of `otelcol_receiver_refused_spans` and
`otelcol_receiver_refused_metric_points` indicate too many errors returned to
clients. Depending on the deployment and the client’s resilience this may
indicate data loss at the clients.

Sustained rates of `otelcol_exporter_send_failed_spans` and
`otelcol_exporter_send_failed_metric_points` indicate that the Collector is not
able to export data as expected.
It doesn't imply data loss per se since there could be retries but a high rate
of failures could indicate issues with the network or backend receiving the
data.

## Data Flow

### Data Ingress

The `otelcol_receiver_accepted_spans` and
`otelcol_receiver_accepted_metric_points` metrics provide information about
the data ingested by the Collector.

### Data Egress

The `otecol_exporter_sent_spans` and
`otelcol_exporter_sent_metric_points`metrics provide information about
the data exported by the Collector.
