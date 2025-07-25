# Build stage for compiling the collector
FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY . .

# Install build dependencies
RUN apk add --no-cache git make

# Build the collector binary for the target architecture
WORKDIR /src/cmd/otelcorecol
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /src/otelcol \
    -tags "" .

# Certificate stage
FROM alpine:3.16 as certs
RUN apk --update add ca-certificates

# Final stage
FROM scratch

ARG USER_UID=10001
USER ${USER_UID}

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder --chmod=755 /src/otelcol /otelcol
COPY configs/otelcol.yaml /etc/otelcol/config.yaml
ENTRYPOINT ["/otelcol"]
CMD ["--config", "/etc/otelcol/config.yaml"]
EXPOSE 4317 55678 55679 