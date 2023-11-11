package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"path/filepath"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	helloworldpb "github.com/ssengalanto/grpc-gateway/proto"
)

type server struct {
	helloworldpb.UnimplementedGreeterServer
}

func NewServer() *server {
	return &server{}
}

func (s *server) SayHello(ctx context.Context, in *helloworldpb.HelloRequest) (*helloworldpb.HelloReply, error) {
	return &helloworldpb.HelloReply{Message: in.Name + " world"}, nil
}

func main() {
	// Create a listener on TCP port
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create a gRPC server object
	s := grpc.NewServer()
	// Attach the Greeter service to the server
	helloworldpb.RegisterGreeterServer(s, &server{})

	// Serve gRPC server
	log.Println("Serving gRPC on 0.0.0.0:8080")
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Create a client connection to the gRPC server we just started
	// This is where the gRPC-Gateway proxies the requests
	conn, err := grpc.DialContext(
		context.Background(),
		"0.0.0.0:8080",
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to dial server: %v", err)
	}

	gwmux := runtime.NewServeMux()
	// Register Greeter
	if err := helloworldpb.RegisterGreeterHandler(context.Background(), gwmux, conn); err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}

	// Serve the Swagger JSON file
	swaggerFile := filepath.Join(".", "gen", "openapiv2", "service.swagger.json")

	// mount a path to expose the generated OpenAPI specification on disk
	gwmux.HandlePath("GET", "/docs/swagger.json", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		http.ServeFile(w, r, swaggerFile)
	})

	// mount the Swagger UI that uses the OpenAPI specification path above
	gwmux.HandlePath("GET", "/docs/*", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		http.StripPrefix("/docs", http.FileServer(http.Dir("./swagger-ui"))).ServeHTTP(w, r)
	})

	gwServer := &http.Server{
		Addr:    ":8090",
		Handler: gwmux,
	}

	log.Println("Serving gRPC-Gateway on http://0.0.0.0:8090")
	if err := gwServer.ListenAndServe(); err != nil {
		log.Fatalf("Failed to serve gRPC-Gateway: %v", err)
	}
}
