# Define a default value so it's not empty if the builder fails to provide it
ARG BUILDPLATFORM=linux/amd64

FROM --platform=$BUILDPLATFORM golang:1.25.6-alpine AS builder
ARG TARGETOS=linux
ARG TARGETARCH=amd64
WORKDIR /src

# restore dependencies
COPY go.mod go.sum ./
COPY . .

# Skaffold passes in debug-oriented compiler flags
ARG SKAFFOLD_GO_GCFLAGS
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -mod=vendor -ldflags="-s -w" -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o /frontend .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates busybox-extras net-tools bind-tools

WORKDIR /src
COPY --from=builder /frontend /src/frontend
COPY --from=builder /src/templates /src/templates
COPY --from=builder /src/static /src/static

# Definition of this variable is used by 'skaffold debug' to identify a golang binary.
# Default behavior - a failure prints a stack trace for the current goroutine.
# See https://golang.org/pkg/runtime/
ENV GOTRACEBACK=single

EXPOSE 8080
ENTRYPOINT ["/src/frontend"]
