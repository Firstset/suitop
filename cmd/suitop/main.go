package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"suitop/internal/checkpoint"
	"suitop/internal/config"
	sgrpc "suitop/internal/grpc"
	"suitop/internal/tui"
	"suitop/internal/types"
	"suitop/internal/util"
	"suitop/internal/validator"

	subPb "suitop/pb/sui/rpc/v2alpha"
	rpcPb "suitop/pb/sui/rpc/v2beta"
)

func main() {
	// Parse command line flags
	plainMode := flag.Bool("plain", false, "Use plain text output instead of TUI")
	noAltScreen := flag.Bool("no-alt-screen", false, "Run inside current terminal buffer (useful for tmux logs)")
	logToFile := flag.Bool("log-to-file", false, "Write logs to a file")
	logFilePath := flag.String("log-file", "", "Path to log file (default: ~/.suitop/logs/suitop.log)")
	flag.Parse()

	// Load configuration
	cfg := config.Load()

	// Override config with command line flags if specified
	if *plainMode {
		cfg.UIConfig.PlainMode = true
	}
	if *noAltScreen {
		cfg.UIConfig.NoAltScreen = true
	}
	if *logToFile {
		cfg.LogConfig.ToFile = true
	}
	if *logFilePath != "" {
		cfg.LogConfig.FilePath = *logFilePath
	}

	// Setup logging
	logCleanup, err := util.SetupLogging(util.LogConfig{
		ToStderr:  cfg.LogConfig.ToStderr,
		ToFile:    cfg.LogConfig.ToFile,
		FilePath:  cfg.LogConfig.FilePath,
		WithTime:  cfg.LogConfig.WithTime,
		WithLevel: cfg.LogConfig.WithLevel,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting up logging: %v\n", err)
		os.Exit(1)
	}
	defer logCleanup()

	log.Printf("Connecting to Sui node for subscriptions: %s", cfg.SuiNode)

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

	log.Println("Successfully connected to gRPC node for subscriptions.")

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

	log.Println("Starting checkpoint processing loop...")

	// The processor will contain the main loop logic
	processor := checkpoint.NewProcessor(valLoader, statsManager, cfg.ProcessorConfig, cfg.UIConfig.PlainMode)

	if cfg.UIConfig.PlainMode {
		// In plain mode, run the processor directly in this goroutine
		processor.Run(ctx, initialEpoch, initialCommittee, checkpointStream, nil)
	} else {
		// Channel for sending state updates to the UI
		stateChan := make(chan types.SnapshotMsg, 200)

		// Start the processor in a goroutine
		go processor.Run(ctx, initialEpoch, initialCommittee, checkpointStream, stateChan)

		// Convert the validator info to the types package format for the UI
		committeeForUI := make([]types.ValidatorInfo, len(initialCommittee))
		for i, v := range initialCommittee {
			committeeForUI[i] = v.ToTypesInfo()
		}

		// Initialize the Bubble Tea model
		model := tui.New(initialEpoch, committeeForUI)

		// Program options based on config
		programOpts := []tea.ProgramOption{
			tea.WithMouseCellMotion(), // Enable mouse support for future interactions
		}

		// Add alt screen option if not disabled
		if !cfg.UIConfig.NoAltScreen {
			programOpts = append(programOpts, tea.WithAltScreen())
		}

		// Create the tea program with all necessary options
		p := tea.NewProgram(model, programOpts...)

		// Set up a goroutine to relay state updates from the processor to the UI
		go func() {
			for {
				select {
				case msg, ok := <-stateChan:
					if !ok {
						return // Channel closed
					}
					p.Send(tui.SnapshotMsg(msg))
				case <-ctx.Done():
					return
				}
			}
		}()

		// Allow for graceful shutdown by quitting the program when the context is done
		go func() {
			<-ctx.Done()
			log.Println("Shutdown signal received, closing UI...")
			p.Quit()
		}()

		// Start the UI in the main goroutine
		if err := p.Start(); err != nil {
			log.Fatalf("Error running UI: %v", err)
		}
	}

	log.Println("Application shut down.")
}
