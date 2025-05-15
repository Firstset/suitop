package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Config holds all configuration for the application.
type Config struct {
	SuiNode              string
	JSONRPCURL           string
	DefaultRPCTimeout    time.Duration
	GRPC                 GRPCConfig
	GRPCSubscriberConfig GRPCSubscriberConfig // Renamed for clarity
	ProcessorConfig      ProcessorConfig      // Placeholder for processor specific configs
	RPCClientConfig      RPCClientConfig      // For RPC client settings passed to validator.Loader
	UIConfig             UIConfig             // For UI related settings
	LogConfig            LogConfig            // For logging configuration
}

// GRPCConfig holds gRPC specific settings.
type GRPCConfig struct {
	UseTLS             bool
	InsecureSkipVerify bool
	// Other gRPC dial options can be added here
}

// GRPCSubscriberConfig holds settings for the checkpoint subscriber.
type GRPCSubscriberConfig struct {
	RetryDelay time.Duration
	// MaxRetries int // Example: could add max retries or backoff strategy config
}

// ProcessorConfig can hold settings for the checkpoint processor if needed.
type ProcessorConfig struct {
	// Example: BatchSize int, etc.
}

// RPCClientConfig holds settings for the JSON-RPC client.
type RPCClientConfig struct {
	URL     string
	Timeout time.Duration
}

// UIConfig contains settings for the user interface
type UIConfig struct {
	PlainMode   bool // When true, use command-line output instead of TUI
	NoAltScreen bool // When true, run inside current terminal buffer (useful for tmux logs)
}

// LogConfig holds settings for logging
type LogConfig struct {
	ToStderr  bool   // Whether to log to stderr
	ToFile    bool   // Whether to log to a file
	FilePath  string // Path to log file
	WithTime  bool   // Include timestamps in logs
	WithLevel bool   // Include log levels in messages
}

// Load populates Config from environment variables or defaults.
func Load() *Config {
	suiNode := os.Getenv("SUI_NODE")
	if suiNode == "" {
		suiNode = "fullnode.mainnet.sui.io:443" // Default gRPC node
	}

	jsonRPCURL := os.Getenv("SUI_JSON_RPC_URL")
	if jsonRPCURL == "" {
		jsonRPCURL = "https://fullnode.mainnet.sui.io" // Default JSON-RPC endpoint
	}

	defaultRPCTimeoutStr := os.Getenv("DEFAULT_RPC_TIMEOUT_SECONDS")
	defaultRPCTimeoutSeconds, err := strconv.Atoi(defaultRPCTimeoutStr)
	if err != nil || defaultRPCTimeoutSeconds <= 0 {
		defaultRPCTimeoutSeconds = 15 // Default to 15 seconds
	}

	// Example for GRPC specific config from env (can be expanded)
	grpcUseTLSStr := os.Getenv("GRPC_USE_TLS")
	grpcUseTLS := true // Default to true
	if grpcUseTLSStr == "false" {
		grpcUseTLS = false
	}

	grpcInsecureSkipVerifyStr := os.Getenv("GRPC_INSECURE_SKIP_VERIFY")
	grpcInsecureSkipVerify := true // Default to true for easier local dev against testnets, review for prod
	if grpcInsecureSkipVerifyStr == "false" {
		grpcInsecureSkipVerify = false
	}

	subscriberRetryDelayStr := os.Getenv("SUBSCRIBER_RETRY_DELAY_MS")
	subscriberRetryDelayMs, err := strconv.Atoi(subscriberRetryDelayStr)
	if err != nil || subscriberRetryDelayMs <= 0 {
		subscriberRetryDelayMs = 1000 // Default to 1 second
	}

	// UI Config settings
	plainModeStr := os.Getenv("PLAIN_MODE")
	plainMode := false // Default to TUI mode
	if plainModeStr == "true" {
		plainMode = true
	}

	noAltScreenStr := os.Getenv("NO_ALT_SCREEN")
	noAltScreen := false // Default to using alt screen
	if noAltScreenStr == "true" {
		noAltScreen = true
	}

	// Log Config settings
	logToFileStr := os.Getenv("LOG_TO_FILE")
	logToFile := false
	if logToFileStr == "true" {
		logToFile = true
	}

	logFilePath := os.Getenv("LOG_FILE_PATH")
	if logFilePath == "" {
		// Default log file in user's home directory
		home, err := os.UserHomeDir()
		if err == nil {
			logFilePath = filepath.Join(home, ".suitop", "logs", "suitop.log")
		} else {
			// Fallback to current directory if home not found
			logFilePath = "suitop.log"
		}
	}

	return &Config{
		SuiNode:           suiNode,
		JSONRPCURL:        jsonRPCURL,
		DefaultRPCTimeout: time.Duration(defaultRPCTimeoutSeconds) * time.Second,
		GRPC: GRPCConfig{
			UseTLS:             grpcUseTLS,
			InsecureSkipVerify: grpcInsecureSkipVerify,
		},
		GRPCSubscriberConfig: GRPCSubscriberConfig{
			RetryDelay: time.Duration(subscriberRetryDelayMs) * time.Millisecond,
		},
		ProcessorConfig: ProcessorConfig{},
		RPCClientConfig: RPCClientConfig{
			URL:     jsonRPCURL,
			Timeout: time.Duration(defaultRPCTimeoutSeconds) * time.Second,
		},
		UIConfig: UIConfig{
			PlainMode:   plainMode,
			NoAltScreen: noAltScreen,
		},
		LogConfig: LogConfig{
			ToStderr:  true,      // Always log to stderr
			ToFile:    logToFile, // Optionally log to file
			FilePath:  logFilePath,
			WithTime:  true,
			WithLevel: true,
		},
	}
}
