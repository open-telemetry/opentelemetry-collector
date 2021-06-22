job "otel-collector" {
  datacenters = ["dc1"]
  type        = "service"

  group "otel-collector" {
    count = 1

    network {
      port "metrics" {
        to = 8888
      }

      # Receivers
      port "otlp" {
        to = 4317
      }

      port "jaeger-grpc" {
        to = 14250
      }

      port "jaeger-thrift-http" {
        to = 14268
      }

      port "zipkin" {
        to = 9411
      }

      # Extensions
      port "health-check" {
        to = 13133
      }

      port "zpages" {
        to = 55679
      }
    }

    service {
      name = "otel-collector"
      port = "health-check"
      tags = ["health"]

      check {
        type     = "http"
        port     = "health-check"
        path     = "/"
        interval = "5s"
        timeout  = "2s"
      }
    }

    service {
      name = "otel-collector"
      port = "otlp"
      tags = ["otlp"]
    }

    service {
      name = "otel-collector"
      port = "jaeger-grpc"
      tags = ["jaeger-grpc"]
    }

    service {
      name = "otel-collector"
      port = "jaeger-thrift-http"
      tags = ["jaeger-thrift-http"]
    }

    service {
      name = "otel-collector"
      port = "zipkin"
      tags = ["zipkin"]
    }

    service {
      name = "otel-agent"
      port = "metrics"
      tags = ["metrics"]
    }

    service {
      name = "otel-agent"
      port = "zpages"
      tags = ["zpages"]
    }

    task "otel-collector" {
      driver = "docker"

      config {
        image = "otel/opentelemetry-collector-dev:latest"

        entrypoint = [
          "/otelcol",
          "--config=local/config/otel-collector-config.yaml",
          # Memory Ballast size should be max 1/3 to 1/2 of memory.
          "--mem-ballast-size-mib=683"
        ]

        ports = [
          "metrics",
          "otlp",
          "jaeger-grpc",
          "jaeger-thrift-http",
          "zipkin",
          "health-check",
          "zpages",
        ]
      }

      resources {
        cpu    = 500
        memory = 2048
      }

      template {
        data        = <<EOF
receivers:
  otlp:
    protocols:
      grpc:
      http:
  jaeger:
    protocols:
      grpc:
      thrift_http:
  zipkin: {}
processors:
  batch:
  memory_limiter:
    # Same as --mem-ballast-size-mib CLI argument
    ballast_size_mib: 683
    # 80% of maximum memory up to 2G
    limit_mib: 1500
    # 25% of limit up to 2G
    spike_limit_mib: 512
    check_interval: 5s
extensions:
  health_check: {}
  zpages: {}
exporters:
  zipkin:
    endpoint: "http://somezipkin.target.com:9411/api/v2/spans" # Replace with a real endpoint.
  jaeger:
    endpoint: "somejaegergrpc.target.com:14250" # Replace with a real endpoint.
    insecure: true
service:
  extensions: [health_check, zpages]
  pipelines:
    traces/1:
      receivers: [otlp, zipkin]
      processors: [memory_limiter, batch]
      exporters: [zipkin]
    traces/2:
      receivers: [otlp, jaeger]
      processors: [memory_limiter, batch]
      exporters: [jaeger]
EOF
        destination = "local/config/otel-collector-config.yaml"
      }
    }
  }
}
