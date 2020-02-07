FROM golang:1.13 as build
WORKDIR /src

ADD . .
RUN go mod download
RUN make install-tools
RUN make otelcol

FROM alpine:3.10
COPY --from=build /src/bin/linux/otelcol /otelcol

EXPOSE 55678 55679
ENTRYPOINT ["/otelcol"]
