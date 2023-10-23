FROM golang:1.20.0

RUN apt-get update && apt-get install -y build-essential

RUN git clone https://github.com/open-telemetry/opentelemetry-proto-profile.git /opentelemetry-proto && \
    cd /opentelemetry-proto && git checkout 622c1658673283102a9429109185615bfcfaa78e

RUN git clone https://github.com/petethepig/opentelemetry-collector.git /opentelemetry-collector && \
    cd /opentelemetry-collector && git checkout 2e4260da77665efd9d2111b4ec10a15331d18d27

WORKDIR /opentelemetry-collector

RUN make otelcorecol
RUN mv ./bin/otelcorecol_*_* ./bin/otelcorecol

ENTRYPOINT [ "/opentelemetry-collector/bin/otelcorecol" ]
