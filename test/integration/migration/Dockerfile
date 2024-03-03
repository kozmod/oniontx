FROM golang:1.22-alpine

RUN --mount=type=cache,target=/var/cache/apk apk --update --upgrade add ca-certificates gcc libc-dev

ENV GO111MODULE on
ENV CGO_ENABLED 1
ENV GOOS linux

WORKDIR /go/migration
ADD . /go/migration
RUN go build -o migration .
