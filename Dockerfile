FROM golang:1.11.4-alpine3.7 as builder

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh gcc libc-dev

ENV GOPKG github.com/census-instrumentation/opencensus-service
COPY . /go/src/$GOPKG/

RUN cd /go/src/$GOPKG/ && ./build_binaries.sh linux && cp /go/src/$GOPKG/bin/ocagent_linux /ocagent

FROM alpine:3.7

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh

COPY --from=builder /ocagent /ocagent

# Expose the OpenCensus receiver port.
EXPOSE 55678/tcp

CMD ["/ocagent"]
