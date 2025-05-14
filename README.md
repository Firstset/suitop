# Sui Uptime Monitor

Monitors validator uptime on the Sui network by subscribing to checkpoint data.

## Configuration

The application can be configured using environment variables:

- `SUI_NODE`: The gRPC endpoint for Sui node subscriptions (e.g., `fullnode.mainnet.sui.io:443`).
- `SUI_JSON_RPC_URL`: The JSON-RPC endpoint for Sui fullnode (e.g., `https://fullnode.mainnet.sui.io`).
- `DEFAULT_RPC_TIMEOUT_SECONDS`: Timeout for JSON-RPC calls in seconds (default: 15).
- `GRPC_USE_TLS`: Set to `true` or `false` to enable/disable TLS for gRPC (default: `true`).
- `GRPC_INSECURE_SKIP_VERIFY`: Set to `true` or `false` to skip TLS certificate verification for gRPC (default: `true`).
- `SUBSCRIBER_RETRY_DELAY_MS`: Delay in milliseconds before retrying gRPC subscription (default: 1000).

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
./suitop
```

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