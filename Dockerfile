FROM alpine:latest as prep
RUN apk --update add ca-certificates

RUN mkdir -p /tmp

FROM scratch

ARG USER_UID=10001
USER ${USER_UID}

COPY --from=prep /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY bin/otelcorecol_linux_amd64 /otelcorecol
EXPOSE 4317 55680 55679
ENTRYPOINT ["/otelcorecol"]
CMD ["--config", "/etc/otel/config.yaml"]
