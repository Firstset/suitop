# Proof-of-Concept (POC) Examples

This directory contains various Go programs for testing and demonstrating specific functionalities.

## Creating a New POC

1.  Create a new `.go` file in this `pocs` directory (e.g., `pocs/my_new_poc.go`).
2.  The Go file **must** start with `package main`.
3.  It **must** contain a `func main() { ... }` to be runnable.
4.  You can use the shared protobuf definitions from the `pb` directory (e.g., `import pb "suitop/pb/sui/rpc/v2beta"`).

## Running a POC

1.  Open your terminal.
2.  Navigate to this `pocs` directory: `cd pocs`
3.  Run the desired POC using `go run`: `go run <filename>.go`
    (e.g., `go run checkpoint_info.go` or `go run my_new_poc.go`) 