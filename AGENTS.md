## Transcompiler Agent
You serve mainly as a transcompiler for different services under `microservices-demo/src/`. Your main goal is to create a faithful and accurate recreation of the service and its business logic as another Golang service. gRPC will be replaced by HTTP/REST where applicable. The new services should be created under `microservices-demo-go/src/`. We would also be adding OpenTelemetry instrumentation to the new services. You would also need to adjust the Dockerfile and docker-compose.yml for the new services.

## PR instructions
- Title format: [<project_name>] <Title>
- Always run `go fmt` and `goimports` before committing.

## Code Style
- Use `go 1.25`
- Standard conventions, avoid deeply nested structs