FROM alpine:3.12 as certs
RUN apk --update add ca-certificates
RUN mkdir -m 777 /scratchtmp

FROM alpine:3.12 AS otelcol
COPY otelcol /
# Note that this shouldn't be necessary, but in some cases the file seems to be
# copied with the execute bit lost (see #1317)
RUN chmod 755 /otelcol

FROM scratch
# Make an empty /tmp directory as the Kubernetes client-go library tries to log to here.
# If it is unable to do so, it causes the collector to exit (see issue #587).
COPY --from=certs /scratchtmp /tmp
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=otelcol /otelcol /
ENTRYPOINT ["/otelcol"]
EXPOSE 55678 55679
