FROM alpine:3.9 as certs
RUN apk --update add ca-certificates
RUN mkdir -m 777 /scratchtmp

FROM scratch
# Make an empty /tmp directory as the Kubernetes client-go library tries to log to here.
# If it is unable to do so, it causes the collector to exit (see issue #587).
COPY --from=certs /scratchtmp /tmp
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY otelcol /
ENTRYPOINT ["/otelcol"]
EXPOSE 55678 55679
