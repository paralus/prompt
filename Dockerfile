FROM golang:1.21 as build
LABEL description="Build container"

ENV CGO_ENABLED 0
COPY . /build
WORKDIR /build
RUN go build -ldflags "-s" github.com/paralus/prompt

FROM alpine:latest as runtime
LABEL description="Run container"

COPY --from=build /build/prompt /usr/bin/prompt
WORKDIR /usr/bin
CMD ./prompt

EXPOSE 7009
