FROM golang:alpine AS builder

ARG RESTIC_VERSION=0.9.4

RUN set -x \
  && apk add --update --no-cache ca-certificates

WORKDIR /go/src/github.com/appscodelabs/actions/cluster-tool
COPY ./cluster-tool/ .

# Dowonload and extract restic binary
RUN set -x \
    && wget https://github.com/restic/restic/releases/download/v${RESTIC_VERSION}/restic_${RESTIC_VERSION}_linux_amd64.bz2 \
    && bzip2 -d ./restic_${RESTIC_VERSION}_linux_amd64.bz2 \
    && mv ./restic_${RESTIC_VERSION}_linux_amd64 ./restic

# Build cluster-tool binary
RUN CGO_ENABLED=0 GOOS=linux go build -o ./cluster-tool ./main.go


# Build final image
FROM alpine:3.8
RUN set -x \
  && apk add --update --no-cache ca-certificates

COPY --from=builder /go/src/github.com/appscodelabs/actions/cluster-tool/cluster-tool /bin/cluster-tool
COPY --from=builder /go/src/github.com/appscodelabs/actions/cluster-tool/restic /bin/restic
ENTRYPOINT ["/bin/cluster-tool"]
