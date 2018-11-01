# Use a helper to create an emtpy configuration since the agent requires such file
FROM alpine:3.7 as helper
RUN touch ./config.yaml

FROM scratch
COPY ocagent_linux /
COPY --from=helper ./config.yaml config.yaml
ENTRYPOINT ["/ocagent_linux"]
EXPOSE 55678 55679
