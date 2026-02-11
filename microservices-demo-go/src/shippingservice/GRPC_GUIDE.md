# How shippingservice Works (gRPC Guide)

## What is gRPC (quick primer)

gRPC is a framework for **remote procedure calls** -- it lets one program call a function on another program over the network, as if it were a local function call. The key pieces are:

1. **Protocol Buffers (protobuf)** -- A language for defining your API contract: what methods exist, and what the request/response messages look like. This is defined in a `.proto` file.
2. **Code generation** -- A tool (`protoc`) reads the `.proto` file and generates Go (or other language) code: strongly typed structs for messages, and interfaces for services.
3. **Server** -- Your code implements the generated interface (the actual business logic).
4. **Client** -- Other services use the generated client code to call your server over the network.

Instead of REST (where you'd do `POST /shipOrder` with a JSON body), gRPC uses a binary protocol (HTTP/2 + protobuf) which is faster and gives you type-safe contracts.

---

## How shippingservice is structured

The service has **4 files** that matter:

### 1. The generated protobuf code (`genproto/`)

These files were auto-generated from a `demo.proto` file. They define the **contract** -- the `ShippingService` has two RPC methods:

```go
type ShippingServiceServer interface {
    GetQuote(context.Context, *GetQuoteRequest) (*GetQuoteResponse, error)
    ShipOrder(context.Context, *ShipOrderRequest) (*ShipOrderResponse, error)
}
```

This is the interface your server **must implement**. In gRPC terms:

| RPC Method | Request Message | Response Message | Purpose |
|---|---|---|---|
| `GetQuote` | `GetQuoteRequest` (contains items + address) | `GetQuoteResponse` (contains `Money`) | Get a shipping cost estimate |
| `ShipOrder` | `ShipOrderRequest` (contains items + address) | `ShipOrderResponse` (contains `TrackingId`) | "Ship" items and get a tracking ID |

The generated code also provides `UnimplementedShippingServiceServer` -- a struct with default "not implemented" responses. Your server **embeds** this so that if new methods are added to the proto in the future, your code won't break (it'll just return "unimplemented" for those).

### 2. `main.go` -- The server setup and RPC implementations

This is where all the pieces come together.

**Server startup:**

```go
func main() {
    // ... tracing/profiling setup omitted ...

    port := defaultPort  // "50051"
    // ... env var override ...

    lis, err := net.Listen("tcp", port)      // Step 1: Open a TCP socket
    srv = grpc.NewServer()                    // Step 2: Create a gRPC server
    svc := &server{}                          // Step 3: Create your service impl
    pb.RegisterShippingServiceServer(srv, svc) // Step 4: Register it with the gRPC server
    // ... health check registration ...
    reflection.Register(srv)                  // Step 5: Enable reflection (for debugging tools)
    srv.Serve(lis)                            // Step 6: Start serving!
}
```

Here's the flow:
1. **Listen** on a TCP port (default `50051`)
2. **Create** a `grpc.Server`
3. **Register** your service implementation with that server (this tells gRPC "when someone calls `GetQuote`, route it to my `server.GetQuote` method")
4. **Serve** -- blocks forever, handling incoming gRPC calls

**The `server` struct:**

```go
type server struct {
    pb.UnimplementedShippingServiceServer
}
```

By embedding `UnimplementedShippingServiceServer`, this struct satisfies the `ShippingServiceServer` interface. Then you **override** the methods you want to implement:

**`GetQuote` implementation:**

```go
func (s *server) GetQuote(ctx context.Context, in *pb.GetQuoteRequest) (*pb.GetQuoteResponse, error) {
    log.Info("[GetQuote] received request")
    defer log.Info("[GetQuote] completed request")

    quote := CreateQuoteFromCount(0)

    return &pb.GetQuoteResponse{
        CostUsd: &pb.Money{
            CurrencyCode: "USD",
            Units:        int64(quote.Dollars),
            Nanos:        int32(quote.Cents * 10000000)},
    }, nil
}
```

When a client calls `GetQuote`, gRPC deserializes the protobuf request into a `*pb.GetQuoteRequest`, calls this method, and serializes the returned `*pb.GetQuoteResponse` back to the client. Note: this implementation currently ignores the input items and always returns a hardcoded $8.99 quote.

**`ShipOrder` implementation:**

```go
func (s *server) ShipOrder(ctx context.Context, in *pb.ShipOrderRequest) (*pb.ShipOrderResponse, error) {
    log.Info("[ShipOrder] received request")
    defer log.Info("[ShipOrder] completed request")

    baseAddress := fmt.Sprintf("%s, %s, %s", in.Address.StreetAddress, in.Address.City, in.Address.State)
    id := CreateTrackingId(baseAddress)

    return &pb.ShipOrderResponse{
        TrackingId: id,
    }, nil
}
```

This reads the address from the request, generates a random tracking ID, and returns it. It's a mock -- nothing is actually shipped.

### 3. `quote.go` -- Business logic for pricing

```go
func CreateQuoteFromCount(count int) Quote {
    return CreateQuoteFromFloat(8.99)
}
```

Always returns `$8.99` regardless of the item count. A real implementation would calculate based on weight, distance, etc.

### 4. `tracker.go` -- Tracking ID generation

```go
func CreateTrackingId(salt string) string {
    if !seeded {
        rand.Seed(time.Now().UnixNano())
        seeded = true
    }

    return fmt.Sprintf("%c%c-%d%s-%d%s",
        getRandomLetterCode(),
        getRandomLetterCode(),
        len(salt),
        getRandomNumber(3),
        len(salt)/2,
        getRandomNumber(7),
    )
}
```

Generates IDs like `"AB-15432-72948371"` using randomness seeded from the address string length.

---

## How a client would call this service

Another microservice (say, a checkout service) would call shippingservice like this (conceptually):

```go
// 1. Connect to the shipping service
conn, _ := grpc.Dial("shippingservice:50051", grpc.WithInsecure())
defer conn.Close()

// 2. Create a client from the generated code
client := pb.NewShippingServiceClient(conn)

// 3. Call the RPC method like a normal function!
resp, err := client.GetQuote(ctx, &pb.GetQuoteRequest{
    Address: &pb.Address{...},
    Items:   []*pb.CartItem{...},
})
// resp.CostUsd contains the shipping cost
```

The generated client code (in `demo_grpc.pb.go`) handles all the serialization and network transport for you.

---

## Summary

| Concept | Where in this codebase |
|---|---|
| Service contract (interface) | `genproto/demo_grpc.pb.go` -- `ShippingServiceServer` interface |
| Message types (request/response structs) | `genproto/demo.pb.go` -- `GetQuoteRequest`, `ShipOrderResponse`, `Money`, etc. |
| Server implementation | `main.go` -- `server` struct with `GetQuote()` and `ShipOrder()` methods |
| Server bootstrap (listen + serve) | `main.go` -- `main()` function |
| Business logic | `quote.go` (pricing) and `tracker.go` (tracking IDs) |

The key takeaway: in gRPC you define a contract in `.proto`, generate code from it, and then just implement the generated interface. The framework handles all the networking, serialization, and routing for you.

---

## What does gRPC generate exactly?

### What gRPC generates (you don't write this)

All of this is networking and serialization plumbing -- the stuff that gets data from point A to point B:

- **Message structs** -- Go structs like `GetQuoteRequest`, `ShipOrderResponse`, `Money`, `Address`. These are just typed containers for data that know how to serialize themselves into binary (protobuf) for sending over the wire.
- **A server interface** -- `ShippingServiceServer` with method signatures like `GetQuote(ctx, *GetQuoteRequest) (*GetQuoteResponse, error)`. This is just a contract that says "if you want to be a shipping service, you must have these methods with these exact input/output types."
- **A client** -- `ShippingServiceClient` that other services use. It handles opening a network connection, serializing the request to binary, sending it over HTTP/2, receiving the response, and deserializing it back into a Go struct. The caller just does `client.GetQuote(ctx, req)` and gets back a response as if it were a local function call.
- **Internal routing/wiring** -- The `RegisterShippingServiceServer` function and handler boilerplate that tells the gRPC server "when a `GetQuote` request arrives on the wire, deserialize it and call the `GetQuote` method on the registered server implementation."

**In one sentence: gRPC generates everything needed to move typed data between services over the network.**

### What you implement (the actual service)

All the business logic. gRPC has zero idea what "shipping" means. You write:

- **`GetQuote`** in `main.go` -- the actual logic that decides the shipping cost is $8.99 (lines 119-134)
- **`ShipOrder`** in `main.go` -- the actual logic that builds an address string and creates a tracking ID (lines 138-149)
- **`CreateQuoteFromCount`** in `quote.go` -- the pricing math
- **`CreateTrackingId`** in `tracker.go` -- the ID generation
- **`main()`** -- starting the server, choosing which port, setting up logging

### The boundary

Think of it like a restaurant:

| gRPC's job (generated) | Your job (implemented) |
|---|---|
| The waiter who takes the order to the kitchen and brings the plate back | The chef who actually cooks the food |
| Knows how to carry a `GetQuoteRequest` from client to server | Knows what to do with that request |
| Knows how to send a `GetQuoteResponse` back | Knows what values to put in that response |
| Defines the menu format (interface) | Decides what's on the menu (business logic) |

So yes -- your understanding is correct. gRPC handles the request/response transport (serialization, deserialization, networking, HTTP/2) and gives you a typed interface so both client and server agree on the shape of the data. Everything else -- what the service actually does -- is yours to write.
