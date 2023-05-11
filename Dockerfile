FROM golang:1.20-alpine AS builder

# override with: --build-args VERSION=1.0.0
ARG VERSION=dev

WORKDIR /go/src/github.com/kacheio/kache
COPY . /go/src/github.com/kacheio/kache

RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags=-X=main.version=${VERSION} -o dist/kache cmd/kache/main.go

## -- IMAGE

FROM golang:latest

COPY --from=builder /go/src/github.com/kacheio/kache/dist/kache .
COPY --from=builder /go/src/github.com/kacheio/kache/kache.sample.yml .

EXPOSE 80

ENTRYPOINT ["./kache"]
CMD ["-config.file=kache.sample.yml"]