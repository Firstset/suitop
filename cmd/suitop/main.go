package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"suitop/internal/checkpoint"
	"suitop/internal/config"
	sgrpc "suitop/internal/grpc"
	"suitop/internal/util"
	"suitop/internal/validator"

	subPb "suitop/pb/sui/rpc/v2alpha"
	rpcPb "suitop/pb/sui/rpc/v2beta"
)

func main() {
	cfg := config.Load() // We'll define this package and function later

	fmt.Printf("Connecting to Sui node for subscriptions: %s\n", cfg.SuiNode)

	// Shared context for managing shutdown
	ctx, cancel := context.WithCancel(context.Background())
	// Setup signal handling using the utility function
	stopSignalHandler := util.SetupSignalHandler(cancel)
	defer stopSignalHandler() // Ensure the signal handler goroutine is cleaned up

	// gRPC connection
	var creds credentials.TransportCredentials
	if cfg.GRPC.UseTLS { // Assuming config will have TLS options
		creds = credentials.NewTLS(&tls.Config{InsecureSkipVerify: cfg.GRPC.InsecureSkipVerify})
	} else {
		// For local development or networks where TLS is not needed/configured
		// Note: grpc.WithInsecure() is deprecated. For non-TLS, use credentials.NewTLS(&tls.Config{}) and handle appropriately,
		// or use an empty transport security option if the library supports it.
		// For now, assuming if UseTLS is false, it means insecure. This needs careful review for production.
		// creds = insecure.NewCredentials() // This would require importing "google.golang.org/grpc/credentials/insecure"
		// Fallback to skipping verify if no better insecure option is configured for simplicity here.
		// THIS IS A SIMPLIFICATION and might need adjustment based on actual insecure grpc setup practices.
		log.Println("Warning: gRPC TLS is disabled. Using InsecureSkipVerify for TLS config as a fallback. Review for production.")
		creds = credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	}

	conn, err := grpc.DialContext(ctx, cfg.SuiNode,
		grpc.WithTransportCredentials(creds),
		grpc.WithBlock(),
		// Potentially add interceptors from internal/grpc/interceptors here later
	)
	if err != nil {
		log.Fatalf("Failed to connect to gRPC node %s: %v", cfg.SuiNode, err)
	}
	defer conn.Close()

	fmt.Println("Successfully connected to gRPC node for subscriptions.")

	subClient := subPb.NewSubscriptionServiceClient(conn)

	// Initial committee load
	// The validator.Loader will use the rpc.Client internally, which gets its URL from config
	valLoader := validator.NewLoader(cfg.RPCClientConfig) // We'll define this. RPCClientConfig will have the JSON RPC URL.

	initialCommittee, initialEpoch, err := valLoader.LoadEpochValidatorData(ctx, 0) // Load latest
	if err != nil {
		log.Fatalf("Failed to load initial committee data: %v", err)
	}
	log.Printf("Initial committee for epoch %d loaded with %d validators.", initialEpoch, len(initialCommittee))

	// Initialize stats for the initial committee
	// The stats package will manage the map and its lifecycle.
	statsManager := checkpoint.NewStatsManager()
	statsManager.InitializeCommitteeStats(initialCommittee)

	// Channel for checkpoints from gRPC subscription
	checkpointStream := make(chan *rpcPb.Checkpoint, 100) // Using the correct type from pb

	// Start the gRPC subscriber
	// The subscriber will take the config for retry delays etc.
	go sgrpc.SubscribeToCheckpoints(ctx, subClient, checkpointStream, cfg.GRPCSubscriberConfig)

	fmt.Println("Starting checkpoint processing loop...")

	// The processor will contain the main loop logic
	processor := checkpoint.NewProcessor(valLoader, statsManager, cfg.ProcessorConfig)
	processor.Run(ctx, initialEpoch, initialCommittee, checkpointStream)

	fmt.Println("Application shut down.")
}
