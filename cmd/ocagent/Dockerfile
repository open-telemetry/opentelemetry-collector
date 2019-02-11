# Use a helper to create an emtpy configuration since the agent requires such file
FROM alpine:3.7 as helper
RUN apk update \
    && apk add --no-cache ca-certificates \
    && update-ca-certificates \
    && touch ./config.yaml

FROM scratch
COPY ocagent_linux /
COPY --from=helper ./config.yaml config.yaml
COPY --from=helper /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/ocagent_linux"]
EXPOSE 55678 55679
