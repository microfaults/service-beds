FROM golang:1.25.6-alpine AS builder

WORKDIR /src
COPY go.mod ./
# COPY go.sum ./ # creating go.mod manually, go.sum might not exist yet if tidy failed

# Download dependencies
# specific to the new module
# RUN go mod download 
# Since I couldn't run go mod tidy, I might need to run it in the container?
# Yes, inside the container go is available.
# So I should copy everything and run go mod tidy then build.

COPY . .
RUN go mod tidy
RUN go build -o /go/bin/frontend .

FROM alpine:3.21.2
RUN apk add --no-cache ca-certificates busybox-extras net-tools bind-tools
WORKDIR /src
COPY --from=builder /go/bin/frontend /src/frontend
COPY --from=builder /src/templates /src/templates
COPY --from=builder /src/static /src/static

ENV GOTRACEBACK=single

EXPOSE 8080
ENTRYPOINT ["/src/frontend"]
