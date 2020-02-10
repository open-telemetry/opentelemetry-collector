FROM alpine:3.10 as certs
RUN apk --update add ca-certificates


FROM golang:1.13 as build
WORKDIR /src

COPY go.mod .
COPY go.sum .
RUN go mod download

ADD . .
RUN make install-tools
RUN make otelcol

FROM alpine:3.10
COPY --from=build /src/bin/linux/otelcol /otelcol

EXPOSE 55678 55679
ENTRYPOINT ["/otelcol"]
