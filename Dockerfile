FROM debian:jessie

COPY ./bin/ocagent_linux ocagent

# Expose the OpenCensus interceptor port
EXPOSE 55678/tcp

CMD ["ocagent"]
