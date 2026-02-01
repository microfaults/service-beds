package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/GoogleCloudPlatform/microservices-demo/src/cartservice/cartstore"
	pb "github.com/GoogleCloudPlatform/microservices-demo/src/cartservice/proto"
	"github.com/GoogleCloudPlatform/microservices-demo/src/cartservice/service"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "7070"
	}

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	// 1. Start gRPC Server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var store cartstore.CartStore
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr != "" {
		log.Printf("Using RedisCartStore with address: %s", redisAddr)
		store = cartstore.NewRedisCartStore(redisAddr)
	} else {
		log.Println("REDIS_ADDR not set. Using MemoryCartStore.")
		store = cartstore.NewMemoryCartStore()
	}

	svc := service.NewCartService(store)
	s := grpc.NewServer()
	pb.RegisterCartServiceServer(s, svc)
	reflection.Register(s)

	go func() {
		log.Printf("gRPC server listening on :%s", port)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	// 2. Start HTTP Gateway
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// Allow the gateway to talk to the local gRPC server
	err = pb.RegisterCartServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%s", port), opts)
	if err != nil {
		log.Fatalf("failed to register gateway: %v", err)
	}

	log.Printf("HTTP gateway listening on :%s", httpPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", httpPort), mux); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}
