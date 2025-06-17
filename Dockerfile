FROM golang:1.24.4 AS build

WORKDIR /src
COPY ./go.mod ./go.sum ./
RUN go mod download

COPY ./ ./
RUN go mod download

ARG VERSION=dev
ARG COMMIT_HASH
ENV CGO_ENABLED=1

RUN CGO_ENABLED=${CGO_ENABLED} go build -ldflags="-w -X 'main.BuildVersion=${VERSION}' -X 'main.CommitHash=${COMMIT_HASH}' -X 'main.GoVersion=$(go version | awk '{print $3}' | sed 's/^go//')'" -o /jellyporter .

FROM debian:12.9-slim AS final

LABEL maintainer="soerenschneider"
RUN useradd -m -s /bin/bash jellyporter
USER jellyporter
COPY --from=build --chown=jellyporter:jellyporter /jellyporter /jellyporter

ENTRYPOINT ["/jellyporter"]
