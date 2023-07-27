FROM node:16.18.0 as frontend

RUN apt-get update && apt-get install -y build-essential


RUN git clone https://github.com/grafana/pyroscope.git /pyroscope && \
    cd /pyroscope && git checkout cfbb83acb2b0d297364b2fff5e11f0fd4d81b39c

WORKDIR /pyroscope

RUN make frontend/build

FROM golang:1.20.0

RUN apt-get update && apt-get install -y build-essential

RUN git clone https://github.com/open-telemetry/opentelemetry-proto-profile.git /opentelemetry-proto && \
    cd /opentelemetry-proto && git checkout 622c1658673283102a9429109185615bfcfaa78e

RUN git clone https://github.com/petethepig/opentelemetry-collector.git /opentelemetry-collector && \
    cd /opentelemetry-collector && git checkout 2e4260da77665efd9d2111b4ec10a15331d18d27

RUN git clone https://github.com/grafana/pyroscope.git /pyroscope && \
    cd /pyroscope && git checkout cfbb83acb2b0d297364b2fff5e11f0fd4d81b39c

WORKDIR /pyroscope

COPY --from=frontend /pyroscope/public/build ./public/build
RUN make go/bin

ENTRYPOINT [ "/pyroscope/pyroscope" ]
