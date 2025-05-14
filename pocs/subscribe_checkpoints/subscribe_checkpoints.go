package main

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	// Use v2alpha for SubscriptionService
	subPb "suitop/pb/sui/rpc/v2alpha"
	// v2beta is still used for Checkpoint message within the subscription response
	// pb "suitop/pb/sui/rpc/v2beta"
)

func main() {
	node := os.Getenv("SUI_NODE")
	if node == "" {
		node = "fullnode.mainnet.sui.io:443"
	}

	fmt.Printf("Connecting to %s\n", node)

	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	conn, err := grpc.Dial(node,
		grpc.WithTransportCredentials(creds),
		// For streaming, WithBlock can cause issues if the stream takes time to establish
		// grpc.WithBlock(),
		grpc.WithTimeout(10*time.Second), // Increased timeout for initial connection
	)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	client := subPb.NewSubscriptionServiceClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Printf("Received signal: %s, shutting down...\n", sig)
		cancel()
	}()

	fmt.Println("Subscribing to checkpoints...")
	stream, err := client.SubscribeCheckpoints(ctx, &subPb.SubscribeCheckpointsRequest{
		ReadMask: &fieldmaskpb.FieldMask{
			Paths: []string{"signature"},
		},
	})
	if err != nil {
		log.Fatalf("SubscribeCheckpoints: %v", err)
	}

	fmt.Println("Waiting for checkpoints...")
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			log.Println("Stream ended (EOF)")
			break
		}
		if err != nil {
			// Check if the error is due to context cancellation (graceful shutdown)
			if grpc.Code(err) == grpc.Code(context.Canceled) || err == context.Canceled {
				log.Println("Subscription cancelled by client.")
				break
			}
			log.Fatalf("Error receiving checkpoint: %v", err)
		}

		if resp.Checkpoint != nil && resp.Checkpoint.Signature != nil {
			sig := resp.Checkpoint.Signature
			fmt.Printf("Received Checkpoint - Cursor: %d, Epoch: %d, Signature: %s, Bitmap: %s\n",
				resp.GetCursor(), // Use GetCursor to safely access optional field
				sig.Epoch,
				hex.EncodeToString(sig.Signature),
				sig.Bitmap,
			)
			// Optionally print bitmap if needed, like in the other POC
			// bytesBmp := make([]byte, len(sig.Bitmap)*4)
			// for i, word := range sig.Bitmap {
			// 	off := i * 4
			// 	bytesBmp[off] = byte(word)
			// 	bytesBmp[off+1] = byte(word >> 8)
			// 	bytesBmp[off+2] = byte(word >> 16)
			// 	bytesBmp[off+3] = byte(word >> 24)
			// }
			// fmt.Printf("  Bitmap: %s\n", hex.EncodeToString(bytesBmp))
		} else {
			fmt.Printf("Received Checkpoint - Cursor: %d (no signature data)\n", resp.GetCursor())
		}

		// Check context cancellation in the loop as well
		select {
		case <-ctx.Done():
			log.Println("Context done, exiting loop.")
			return
		default:
			// Continue
		}
	}
	log.Println("Exited checkpoint receive loop.")
}
