FROM --platform=$BUILDPLATFORM golang:1.25.6-alpine@sha256:98e6cffc31ccc44c7c15d83df1d69891efee8115a5bb7ede2bf30a38af3e3c92 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src


# restore dependencies
COPY go.mod go.sum ./

COPY . .

# Skaffold passes in debug-oriented compiler flags
ARG SKAFFOLD_GO_GCFLAGS
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -mod=vendor -ldflags="-s -w" -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o /productcatalogservice .

FROM alpine:3.22

WORKDIR /src
COPY --from=builder /productcatalogservice /src/productcatalogservice
COPY products.json /src/products.json

# Definition of this variable is used by 'skaffold debug' to identify a golang binary.
# Default behavior - a failure prints a stack trace for the current goroutine.
# See https://golang.org/pkg/runtime/
ENV GOTRACEBACK=single

ENV PORT=3550
EXPOSE 3550
ENTRYPOINT ["/src/productcatalogservice"]
