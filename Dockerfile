# Stage 1 - build
FROM golang:1.20 AS builder

RUN go version

ARG RELEASE_VERSION

COPY . /go/src/
WORKDIR /go/src
RUN set -Eeux && \
    go mod download && \
    go mod verify

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-w -s -X 'main.Version=${RELEASE_VERSION}'"
RUN go test -cover -v ./...

# Stage 2 - alpine image
FROM alpine:3

COPY --from=builder /go/src/js-admissions-controller /
ENTRYPOINT "/js-admissions-controller"
