FROM alpine:3.10 as certs
RUN apk --update add ca-certificates


FROM golang:1.13 as build
WORKDIR /src

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY Makefile .
RUN make install-tools

ADD . .
ARG target=otelcol
RUN make ${target}


FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /src/bin/linux/${target} /otel

EXPOSE 55678 55679
ENTRYPOINT ["/otel"]
