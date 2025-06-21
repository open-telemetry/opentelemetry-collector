# Dockerfile
# Custom OpenTelemetry Collector to fix duplicate exported_job labels
# Addresses GitHub issue #13079

FROM ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-contrib:0.128.0

# Copy the custom configuration that includes the transform processor
COPY collector-config.yaml /etc/otelcol-contrib/collector-config.yaml

# Expose the standard OpenTelemetry ports
EXPOSE 8888  # Prometheus metrics endpoint
EXPOSE 4317  # OTLP gRPC receiver
EXPOSE 4318  # OTLP HTTP receiver

# Set the custom config as the default
CMD ["--config=/etc/otelcol-contrib/collector-config.yaml"]
