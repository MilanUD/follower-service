// cmd: follower-service/main.go (ili samo main.go u rootu)
package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"database-example/handlers"
	followerpb "database-example/proto/follower"
	"database-example/repo"
	"database-example/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	logger := log.New(os.Stdout, "[follower-service] ", log.LstdFlags)

	// --- Repo sloj (Neo4j) ---
	followerRepo, err := repo.NewFollowerRepository(logger)
	if err != nil {
		logger.Fatal("Failed to connect to Neo4j:", err)
	}
	// ako imaš r.Close(ctx) u repou:
	defer followerRepo.Close(context.Background())

	// --- Service sloj ---
	followSvc := &service.FollowerService{
		FollowerRepo: followerRepo,
	}

	// --- Handler sloj ---
	followHandler := handlers.NewFollowerHandler(followSvc)

	// --- gRPC server ---
	addr := os.Getenv("FOLLOWER_SERVICE_ADDRESS")
	if addr == "" {
		addr = ":50051"
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatal("Failed to listen:", err)
	}

	grpcServer := grpc.NewServer()
	followerpb.RegisterFollowerServiceServer(grpcServer, followHandler)
	reflection.Register(grpcServer)

	go func() {
		logger.Println("Starting gRPC server on", addr)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server error:", err)
		}
	}()

	// --- graceful shutdown ---
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)
	<-stopCh

	logger.Println("Shutting down gRPC server...")
	grpcServer.Stop()
}
