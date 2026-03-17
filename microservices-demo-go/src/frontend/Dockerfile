FROM golang:1.25.6-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY . .

RUN go build -mod=vendor -o /go/bin/frontend .

FROM alpine:3.21.2
RUN apk add --no-cache ca-certificates busybox-extras net-tools bind-tools
WORKDIR /src
COPY --from=builder /go/bin/frontend /src/frontend
COPY --from=builder /src/templates /src/templates
COPY --from=builder /src/static /src/static

ENV GOTRACEBACK=single

EXPOSE 8080
ENTRYPOINT ["/src/frontend"]
