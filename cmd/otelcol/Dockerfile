FROM alpine:3.13 as certs
RUN apk --update add ca-certificates

FROM alpine:3.13 AS otelcol
COPY otelcol /
# Note that this shouldn't be necessary, but in some cases the file seems to be
# copied with the execute bit lost (see #1317)
RUN chmod 755 /otelcol

FROM scratch

ARG USER_UID=10001
USER ${USER_UID}

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=otelcol /otelcol /
COPY config.yaml /etc/otel/config.yaml
ENTRYPOINT ["/otelcol"]
CMD ["--config", "/etc/otel/config.yaml"]
EXPOSE 4317 55678 55679
