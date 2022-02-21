# Build the manager binary
FROM golang:1.14.2 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# setup github user and token
ARG BUILD_USR="nothing"
ARG BUILD_PWD="nothing"

# Create the user and group files that will be used in the running 
# container to run the process as an unprivileged user.
RUN mkdir /user && \
    echo 'nobody:x:65534:65534:nobody:/:' > /user/passwd && \
    echo 'nobody:x:65534:' > /user/group

# Create a netrc file using the credentials specified using --build-arg
RUN printf "machine github.com\n\
    login ${BUILD_USR}\n\
    password ${BUILD_PWD}\n\
    \n\
    machine api.github.com\n\
    login ${BUILD_USR}\n\
    password ${BUILD_PWD}\n"\
    >> /root/.netrc
RUN chmod 600 /root/.netrc

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN GOPRIVATE=github.com/RafaySystems/* go mod download

# Copy the go source
COPY main.go main.go
COPY pkg/ pkg/
COPY debug/ debug/
COPY internal/ internal/
COPY proto/ proto/
COPY rpc/ rpc/


# Build
RUN GOPRIVATE=github.com/RafaySystems/* CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o rafay-prompt main.go


FROM alpine:3.11
WORKDIR /opt/rafay
RUN apk --update add vim && apk add --no-cache ca-certificates

RUN addgroup -S rafay && adduser -S rafay -G rafay -h /opt/rafay

USER rafay

RUN wget https://storage.googleapis.com/kubernetes-release/release/v1.20.0/bin/linux/amd64/kubectl && chmod +x kubectl
RUN wget https://github.com/kubevirt/kubevirt/releases/download/v0.38.1/virtctl-v0.38.1-linux-amd64 && mv virtctl-v0.38.1-linux-amd64 virtctl && chmod +x virtctl

COPY --from=builder /workspace/rafay-prompt .
ENTRYPOINT ["./rafay-prompt"]
