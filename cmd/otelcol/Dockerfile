FROM alpine:3.12 as certs
RUN apk --update add ca-certificates

FROM alpine:3.12 AS otelcol
COPY otelcol /
# Note that this shouldn't be necessary, but in some cases the file seems to be
# copied with the execute bit lost (see #1317)
RUN chmod 755 /otelcol

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=otelcol /otelcol /
COPY config.yaml /etc/otel/config.yaml
ENTRYPOINT ["/otelcol"]
CMD ["--config", "/etc/otel/config.yaml"]
EXPOSE 55678 55679
