FROM alpine:3.9 as certs
RUN apk --update add ca-certificates

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY otelsvc_linux /
ENTRYPOINT ["/otelsvc_linux"]
EXPOSE 55678 55679
