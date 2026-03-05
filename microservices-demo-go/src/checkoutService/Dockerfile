FROM --platform=$BUILDPLATFORM golang:1.25.6-alpine@sha256:98e6cffc31ccc44c7c15d83df1d69891efee8115a5bb7ede2bf30a38af3e3c92 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src

# restore dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Skaffold passes in debug-oriented compiler flags
ARG SKAFFOLD_GO_GCFLAGS
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -ldflags="-s -w" -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o /checkoutservice .

FROM alpine:3.22

WORKDIR /src
COPY --from=builder /checkoutservice /src/checkoutservice

# Definition of this variable is used by 'skaffold debug' to identify a golang binary.
# Default behavior - a failure prints a stack trace for the current goroutine.
# See https://golang.org/pkg/runtime/
ENV GOTRACEBACK=single

# Environment variables for the service
ENV PORT=5050
ENV SHIPPING_SERVICE_ADDR=5051
ENV PRODUCT_CATALOG_SERVICE_ADDR=5052
ENV CART_SERVICE_ADDR=5053
ENV CURRENCY_SERVICE_ADDR=5054
ENV EMAIL_SERVICE_ADDR=5055
ENV PAYMENT_SERVICE_ADDR=5056
ENV KAFKA_BROKER_ADDR=""

EXPOSE 5050
ENTRYPOINT ["/src/checkoutservice"]