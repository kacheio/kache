FROM golang:1.20-alpine AS builder

ARG VERSION=0.0.1

WORKDIR /go/src/github.com/kacheio/kache
COPY . /go/src/github.com/kacheio/kache

RUN go mod download
RUN go build -ldflags=-X=main.version=${VERSION} -o bin/kache cmd/kache/main.go

## -- IMAGE

FROM golang:latest

COPY --from=builder /go/src/github.com/kacheio/kache/bin/kache .
COPY --from=builder /go/src/github.com/kacheio/kache/kache.yml .

EXPOSE 80

ENTRYPOINT ["./kache", "-config.file=kache.yml"]