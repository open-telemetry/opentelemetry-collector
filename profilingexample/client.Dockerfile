FROM debian
FROM golang:1.20.0

RUN apt-get update && apt-get install -y build-essential

RUN git clone https://github.com/open-telemetry/opentelemetry-proto-profile.git /opentelemetry-proto && \
    cd /opentelemetry-proto && git checkout 622c1658673283102a9429109185615bfcfaa78e

RUN git clone https://github.com/petethepig/opentelemetry-collector.git /opentelemetry-collector && \
    cd /opentelemetry-collector && git checkout 2e4260da77665efd9d2111b4ec10a15331d18d27

RUN git clone https://github.com/grafana/pyroscope-golang.git /pyroscope-golang && \
    cd /pyroscope-golang && git checkout 205226e941344ff2561f0bc1db08b6ca869df72f

WORKDIR /pyroscope-golang

RUN go build -o client ./example

ENTRYPOINT [ "/pyroscope-golang/client" ]
