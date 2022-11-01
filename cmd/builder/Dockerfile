ARG GO_VERSION=1.18

FROM golang:${GO_VERSION}-alpine

RUN apk add --no-cache git
WORKDIR /src
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY ./ ./


RUN CGO_ENABLED=0 go build \
    -installsuffix 'static' \
    -o /app ./main.go

CMD ["/app"]