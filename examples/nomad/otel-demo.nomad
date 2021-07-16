job "otel-demo" {
  datacenters = ["dc1"]
  type        = "service"

  # Jaeger
  group "jaeger" {
    network {
      port "http" {
        to     = 16686
        static = 16686
      }

      port "thrift" {
        to = 14268
      }

      port "grpc" {
        to = 14250
      }
    }

    service {
      name = "otel-demo-jaeger"
      port = "http"
      tags = ["http"]

      check {
        type     = "http"
        port     = "http"
        path     = "/"
        interval = "5s"
        timeout  = "2s"
      }
    }

    service {
      name = "otel-demo-jaeger"
      port = "thrift"
      tags = ["thrift"]
    }

    service {
      name = "otel-demo-jaeger"
      port = "grpc"
      tags = ["grpc"]
    }

    task "jaeger-all-in-one" {
      driver = "docker"

      config {
        image = "jaegertracing/all-in-one:latest"
        ports = ["http", "thrift", "grpc"]
      }

      resources {
        cpu    = 200
        memory = 100
      }
    }
  }

  # Zipkin
  group "zipkin" {
    network {
      port "http" {
        to     = 9411
        static = 9411
      }
    }

    service {
      name = "otel-demo-zipkin"
      port = "http"

      check {
        type     = "http"
        port     = "http"
        path     = "/health"
        interval = "5s"
        timeout  = "2s"
      }
    }

    task "zipkin-all-in-one" {
      driver = "docker"

      config {
        image = "openzipkin/zipkin:latest"
        ports = ["http"]
      }

      resources {
        cpu    = 250
        memory = 350
      }
    }
  }

  # Collector
  group "otel-collector" {
    network {
      # Prometheus metrics exposed by the collector
      port "metrics" {
        to     = 8888
        static = 8888
      }

      # Receivers
      port "otlp" {
        to = 4317
      }

      # Extensions
      port "pprof" {
        to     = 1888
        static = 1888
      }

      port "health-check" {
        to     = 13133
        static = 13133
      }

      port "zpages" {
        to     = 55679
        static = 55670
      }

      # Exporters
      port "prometheus" {
        to     = 8889
        static = 8889
      }
    }

    service {
      name = "otel-demo-collector"
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
      name = "otel-demo-collector"
      port = "otlp"
      tags = ["otlp"]
    }

    service {
      name = "otel-demo-collector"
      port = "metrics"
      tags = ["metrics"]
    }

    service {
      name = "otel-demo-collector"
      port = "prometheus"
      tags = ["prometheus"]
    }

    task "otel-collector" {
      driver = "docker"

      config {
        image = "otel/opentelemetry-collector-dev:latest"

        entrypoint = [
          "/otelcol",
          "--config=local/config/otel-collector-config.yaml",
        ]

        ports = [
          "pprof",
          "metrics",
          "prometheus",
          "health-check",
          "otlp",
          "zpages",
        ]
      }

      resources {
        cpu    = 200
        memory = 64
      }

      template {
        data        = <<EOF
receivers:
  otlp:
    protocols:
      grpc:

exporters:
  prometheus:
    endpoint: "0.0.0.0:{{ env "NOMAD_PORT_prometheus" }}"
    namespace: promexample
    const_labels:
      label1: value1
  logging:

  zipkin:
    endpoint: "http://{{ with service "otel-demo-zipkin" }}{{ with index . 0 }}{{ .Address }}:{{ .Port }}{{ end }}{{ end }}/api/v2/spans"
    format: proto

  jaeger:
    endpoint: "{{ with service "grpc.otel-demo-jaeger" }}{{ with index . 0 }}{{ .Address }}:{{ .Port }}{{ end }}{{ end }}"
    insecure: true

processors:
  batch:

extensions:
  health_check:
  pprof:
    endpoint: :{{ env "NOMAD_PORT_pprof" }}
  zpages:
    endpoint: :{{ env "NOMAD_PORT_zpages" }}

service:
  extensions: [pprof, zpages, health_check]
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [logging, zipkin, jaeger]
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [logging, prometheus]
EOF
        destination = "local/config/otel-collector-config.yaml"
      }
    }
  }

  # Agent
  group "otel-agent" {
    network {
      # Prometheus metrics exposed by the agent
      port "metrics" {
        to     = "8888"
        static = "8887"
      }

      # Receivers
      port "jaeger-grpc" {
        to = 14250
      }

      port "jaeger-thrift-http" {
        to = 14268
      }

      port "opencensus" {
        to = 55678
      }

      port "otlp" {
        to = 4317
      }

      port "zipkin" {
        to = 9411
      }

      # Extensions
      port "pprof" {
        to     = 1777
        static = 1777
      }

      port "zpages" {
        to     = 55679
        static = 55679
      }

      port "health-check" {
        to = 13133
      }
    }

    service {
      name = "otel-demo-agent"
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
      name = "otel-demo-agent"
      port = "jaeger-thrift-http"
      tags = ["jaeger-thrift-http"]
    }

    service {
      name = "otel-demo-agent"
      port = "zipkin"
      tags = ["zipkin"]
    }

    task "otel-agent" {
      driver = "docker"

      config {
        image = "otel/opentelemetry-collector-dev:latest"

        entrypoint = [
          "/otelcol",
          "--config=local/config/otel-agent-config.yaml",
        ]

        ports = [
          "metrics",
          "jaeger-grpc",
          "jaeger-thrift-http",
          "opencensus",
          "otlp",
          "zipkin",
          "pprof",
          "zpages",
          "health-check",
        ]

      }

      resources {
        cpu    = 200
        memory = 64
      }

      template {
        data        = <<EOF
receivers:
  otlp:
    protocols:
      grpc:
  opencensus:
  jaeger:
    protocols:
      grpc:
      thrift_http:
  zipkin:

exporters:
  otlp:
    endpoint: "{{ with service "otlp.otel-demo-collector" }}{{ with index . 0 }}{{ .Address }}:{{ .Port }}{{ end }}{{ end }}"
    insecure: true
  logging:
    loglevel: debug

processors:
  batch:

extensions:
  pprof:
    endpoint: :{{ env "NOMAD_PORT_pprof" }}
  zpages:
    endpoint: :{{ env "NOMAD_PORT_zpages" }}
  health_check:

service:
  extensions: [health_check, pprof, zpages]
  pipelines:
    traces:
      receivers: [otlp, opencensus, jaeger, zipkin]
      processors: [batch]
      exporters: [otlp, logging]
    metrics:
      receivers: [otlp, opencensus]
      processors: [batch]
      exporters: [otlp, logging]
EOF
        destination = "local/config/otel-agent-config.yaml"
      }
    }
  }

  # Synthetic load generators
  group "load-generators" {
    task "jaeger-emitter" {
      driver = "docker"

      config {
        image = "omnition/synthetic-load-generator:1.0.25"
      }

      resources {
        cpu    = 200
        memory = 128
      }

      template {
        data        = <<EOF
JAEGER_COLLECTOR_URL = http://{{ with service "jaeger-thrift-http.otel-demo-agent" }}{{ with index . 0 }}{{ .Address }}:{{ .Port }}{{ end }}{{ end }}
EOF
        destination = "local/config.env"
        env         = true
      }
    }

    task "zipkin-emitter" {
      driver = "docker"

      config {
        image = "omnition/synthetic-load-generator:1.0.25"
      }

      resources {
        cpu    = 200
        memory = 128
      }

      template {
        data        = <<EOF
ZIPKINV2_JSON_URL = http://{{ with service "zipkin.otel-demo-agent" }}{{ with index . 0 }}{{ .Address }}:{{ .Port }}{{ end }}{{ end }}/api/v2/spans
EOF
        destination = "local/config.env"
        env         = true
      }
    }
  }

  # Prometheus
  group "prometheus" {
    network {
      port "http" {
        to     = 9090
        static = 9090
      }
    }

    task "prometheus" {
      driver = "docker"

      config {
        image   = "prom/prometheus:latest"
        ports   = ["http"]
        volumes = ["local/config/prometheus.yaml:/etc/prometheus/prometheus.yml"]
      }

      resources {
        cpu    = 100
        memory = 64
      }

      template {
        data        = <<EOF
scrape_configs:
  - job_name: 'otel-collector'
    scrape_interval: 10s
    static_configs:
      - targets: [{{ range $index := service "prometheus.otel-demo-collector" }}'{{ .Address }}:{{ .Port }}',{{ end }}]
      - targets: [{{ range $index := service "metrics.otel-demo-collector" }}'{{ .Address }}:{{ .Port }}',{{ end }}]
EOF
        destination = "local/config/prometheus.yaml"
      }
    }
  }
}
