# Sui Uptime Monitor

Monitors validator uptime on the Sui network by subscribing to checkpoint data. Features a modern terminal user interface (TUI) with real-time updates.

![Suitop Screenshot](./screenshot.png)

## Features

- Real-time monitoring of validator signatures on checkpoints
- Dual progress bars: validator count and voting-power participation per checkpoint
- Interactive TUI with progress bars and formatted tables
- Plain text mode for logging or scripting use cases
- Automatic terminal resizing support
- Graceful shutdown handling for clean exits

## Configuration

The application can be configured using environment variables:

- `SUI_NODE`: The gRPC endpoint for Sui node subscriptions (e.g., `fullnode.mainnet.sui.io:443`).
- `SUI_JSON_RPC_URL`: The JSON-RPC endpoint for Sui fullnode (e.g., `https://fullnode.mainnet.sui.io`).
- `DEFAULT_RPC_TIMEOUT_SECONDS`: Timeout for JSON-RPC calls in seconds (default: 15).
- `GRPC_USE_TLS`: Set to `true` or `false` to enable/disable TLS for gRPC (default: `true`).
- `GRPC_INSECURE_SKIP_VERIFY`: Set to `true` or `false` to skip TLS certificate verification for gRPC (default: `true`).
- `SUBSCRIBER_RETRY_DELAY_MS`: Delay in milliseconds before retrying gRPC subscription (default: 1000).
- `PLAIN_MODE`: Set to `true` to use plain text output instead of TUI (default: `false`).
- `NO_ALT_SCREEN`: Set to `true` to run inside current terminal buffer (default: `false`).
- `LOG_TO_FILE`: Set to `true` to write logs to a file (default: `false`).
- `LOG_FILE_PATH`: Path to log file (default: `~/.suitop/logs/suitop.log`).

## Command-line Flags

These flags override the corresponding environment variables:

- `--plain`: Use plain text output instead of TUI
- `--no-alt-screen`: Run inside current terminal buffer (useful for tmux logs)
- `--log-to-file`: Write logs to a file
- `--log-file [path]`: Path to log file

## Building

```bash
go build -o suitop cmd/suitop/main.go
```

To embed version information:
```bash
go build -ldflags "-X suitop/internal/version.GitCommit=$(git rev-parse HEAD) -X suitop/internal/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X suitop/internal/version.Version=0.1.0" -o suitop cmd/suitop/main.go
```

## Running

```bash
# Run with default TUI mode
./suitop

# Run with plain text output
./suitop --plain

# Run inside current terminal buffer (good for tmux sessions)
./suitop --no-alt-screen

# Run with logging to a file
./suitop --log-to-file --log-file /path/to/logfile.log
```

## Usage

- Press `q` or `Ctrl+C` to quit the application
- Terminal resizing is automatically handled
- Use `SIGINT` (Ctrl+C) or `SIGTERM` for graceful shutdown

## Project Structure

(Details of the project structure as provided in the refactoring request can be added here.)

```
suitop/
├── go.mod
├── go.sum
│
├── cmd/                     
│   └── suitop/
│       └── main.go          
│
├── internal/                
│   ├── config/              
│   │   └── config.go
│   ├── rpc/                 
│   │   ├── client.go        
│   │   ├── committee.go     
│   │   └── systemstate.go   
│   ├── grpc/                
│   │   ├── subscriber.go    
│   │   └── interceptors.go  
│   ├── checkpoint/          
│   │   ├── bitmap.go        
│   │   ├── processor.go     
│   │   └── stats.go         
│   ├── validator/           
│   │   ├── model.go         
│   │   └── loader.go        
│   ├── tui/                 
│   │   ├── messages.go      
│   │   ├── model.go         
│   │   ├── update.go        
│   │   ├── view.go          
│   │   └── style.go         
│   ├── types/               
│   │   └── common.go        
│   ├── util/                
│   │   ├── logger.go        
│   │   ├── retry.go         
│   │   └── signalctx.go     
│   └── version/             
│       └── version.go
│
├── pb/                      
│   ├── sui/rpc/v2alpha/*.go
│   └── sui/rpc/v2beta/*.go
│
└── README.md
``` 