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

var (
	helpFlagVal        *bool
	plainModeFlagVal   *bool
	noAltScreenFlagVal *bool
	logToFileFlagVal   *bool
	logFilePathFlagVal *string
	networkFlagVal     *string
)

func main() {
	// Define flags
	helpFlagVal = flag.Bool("h", false, "Show help message")
	// Bind -help to the same variable as -h for convenience
	flag.BoolVar(helpFlagVal, "help", false, "Show help message (alias for -h)")
	plainModeFlagVal = flag.Bool("plain", false, "Use plain text output (overrides PLAIN_MODE env var)")
	noAltScreenFlagVal = flag.Bool("no-alt-screen", false, "Run inside current terminal buffer (overrides NO_ALT_SCREEN env var, useful for tmux logs)")
	logToFileFlagVal = flag.Bool("log-to-file", false, "Write logs to a file (overrides LOG_TO_FILE env var)")
	networkFlagVal = flag.String("network", "mainnet", "Network to connect to")
	// Default for the flag variable itself. This is used if --log-file is not provided by the user.
	// It's also used as a fallback for TUI mode if no other path is configured.
	logFilePathFlagVal = flag.String("log-file", "./logs/suitop.log", "Path to log file (overrides LOG_FILE_PATH env var")

	// Custom Usage function
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Monitors validator uptime on the Sui network by subscribing to checkpoint data.\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Handle -h or --help
	if *helpFlagVal {
		flag.Usage()
		os.Exit(0)
	}

	// flag.Parse() calls os.Exit(2) after printing usage if an undefined flag is encountered.
	// We check for remaining non-flag arguments.
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: Unrecognized arguments: %v\n\n", flag.Args())
		flag.Usage()
		os.Exit(1) // Use a different exit code for unrecognized arguments vs. undefined flags
	}

	// Load configuration from environment variables and internal defaults
	cfg := config.Load()

	// Override configuration with command-line flags if they were explicitly set
	if flagWasSet("plain") {
		cfg.UIConfig.PlainMode = *plainModeFlagVal
	}
	if flagWasSet("no-alt-screen") {
		cfg.UIConfig.NoAltScreen = *noAltScreenFlagVal
	}
	if flagWasSet("log-to-file") {
		cfg.LogConfig.ToFile = *logToFileFlagVal
	}
	if flagWasSet("log-file") {
		// If --log-file is explicitly set, its value (even if it's the flag's own default path) overrides cfg.
		cfg.LogConfig.FilePath = *logFilePathFlagVal
	}
	// If --log-file was NOT set, cfg.LogConfig.FilePath retains the value from config.Load()
	// (which is from LOG_FILE_PATH env var or config's internal default like ~/.suitop/logs/suitop.log).
	// The flag's own default "./logs/suitop.log" (held in *logFilePathFlagVal if flag not set) is not automatically applied here yet.

	// Special handling for TUI mode logging
	if !cfg.UIConfig.PlainMode { // If current mode is TUI (after considering env vars and --plain flag)
		cfg.LogConfig.ToFile = true    // Force logging to file for TUI
		cfg.LogConfig.ToStderr = false // Don't log to stderr for TUI

		// If LogFilePath is still considered empty or not meaningfully set for TUI mode,
		// and TUI implies logging to file, ensure a path.
		// A common convention is for config.Load() to provide a non-empty default.
		// If LOG_FILE_PATH was set, cfg.LogConfig.FilePath has that.
		// If --log-file was set, cfg.LogConfig.FilePath has that.
		// If neither of those, and config.Load() resulted in an empty string (e.g. no env var and no internal default set by config.Load):
		if cfg.LogConfig.FilePath == "" {
			// Fallback to the default path defined for the --log-file flag itself.
			cfg.LogConfig.FilePath = *logFilePathFlagVal // This is "./logs/suitop.log"
		}
	}

	// Determine SuiNode and JSONRPCURL based on the network flag.
	// This overrides SUI_NODE/SUI_JSON_RPC_URL from env vars or defaults in config.Load().
	// The --network flag has a default of "mainnet", so *networkFlagVal will always be set.
	switch *networkFlagVal {
	case "mainnet":
		cfg.SuiNode = "fullnode.mainnet.sui.io:443"
		cfg.JSONRPCURL = "https://fullnode.mainnet.sui.io"
		cfg.RPCClientConfig.URL = "https://fullnode.mainnet.sui.io"
	case "testnet":
		cfg.SuiNode = "fullnode.testnet.sui.io:443"
		cfg.JSONRPCURL = "https://fullnode.testnet.sui.io"
		cfg.RPCClientConfig.URL = "https://fullnode.testnet.sui.io"
	case "devnet":
		cfg.SuiNode = "fullnode.devnet.sui.io:443"
		cfg.JSONRPCURL = "https://fullnode.devnet.sui.io"
		cfg.RPCClientConfig.URL = "https://fullnode.devnet.sui.io"
	default:
		fmt.Fprintf(os.Stderr, "Error: Invalid --network value '%s'. Must be 'mainnet', 'testnet', or 'devnet'.\n", *networkFlagVal)
		os.Exit(1)
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
		model := tui.New(initialEpoch, committeeForUI, *networkFlagVal)

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

// flagWasSet checks if a flag was explicitly set on the command line.
// It iterates over the flags that were visited (i.e., set).
func flagWasSet(name string) bool {
	wasSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			wasSet = true
		}
	})
	return wasSet
}
