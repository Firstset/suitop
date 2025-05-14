package main

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	// Use the generated Go bindings
	pb "suitop/pb/sui/rpc/v2beta"
)

func main() {
	node := os.Getenv("SUI_NODE")
	if node == "" {
		node = "fullnode.mainnet.sui.io:443"
	}

	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	conn, err := grpc.Dial(node,
		grpc.WithTransportCredentials(creds),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second))
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewLedgerServiceClient(conn)

	ctx := context.Background()

	svcInfo, err := client.GetServiceInfo(ctx, &pb.GetServiceInfoRequest{})
	if err != nil {
		log.Fatalf("GetServiceInfo: %v", err)
	}
	if svcInfo.CheckpointHeight == nil {
		log.Fatalf("GetServiceInfo returned no checkpoint height")
	}
	seq := *svcInfo.CheckpointHeight
	fmt.Printf("Latest checkpoint: %d\n", seq)

	seqNum := seq
	ckpt, err := client.GetCheckpoint(ctx, &pb.GetCheckpointRequest{
		CheckpointId: &pb.GetCheckpointRequest_SequenceNumber{
			SequenceNumber: seqNum,
		},
		ReadMask: &fieldmaskpb.FieldMask{
			Paths: []string{"signature"},
		},
	})
	if err != nil {
		log.Fatalf("GetCheckpoint: %v", err)
	}
	fmt.Printf("Checkpoint: %+v\n", ckpt)

	agg := ckpt.Signature // *pb.ValidatorAggregatedSignature
	if agg == nil {
		log.Fatalf("checkpoint %d has no signature", seq)
	}

	fmt.Println("ValidatorAggregatedSignature:")
	fmt.Printf("  epoch     : %d\n", agg.Epoch)
	fmt.Printf("  signature : %x\n", agg.Signature)

	// Flatten []uint32 bitmap → []byte (little‑endian words)
	bytesBmp := make([]byte, len(agg.Bitmap)*4)
	for i, word := range agg.Bitmap {
		off := i * 4
		bytesBmp[off] = byte(word)
		bytesBmp[off+1] = byte(word >> 8)
		bytesBmp[off+2] = byte(word >> 16)
		bytesBmp[off+3] = byte(word >> 24)
	}
	fmt.Printf("  bitmap    : %s\n", hex.EncodeToString(bytesBmp))
}
