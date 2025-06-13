package grpc

import (
	"context"
	"io"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"suitop/internal/config"

	subPb "suitop/pb/sui/rpc/v2alpha"
	rpcPb "suitop/pb/sui/rpc/v2beta"
)

// SubscribeToCheckpoints subscribes to the checkpoint stream and sends data to a channel.
// It attempts to automatically resubscribe if the stream is terminated.
func SubscribeToCheckpoints(
	ctx context.Context,
	subClient subPb.SubscriptionServiceClient,
	checkpointChan chan<- *rpcPb.Checkpoint, // Changed from rpcPb.Checkpoint to subPb.Checkpoint
	cfg config.GRPCSubscriberConfig,
) {
	defer close(checkpointChan) // Close channel when subscription goroutine exits

	retryDelay := cfg.RetryDelay
	if retryDelay <= 0 {
		retryDelay = 1 * time.Second // Default retry delay if not configured properly
	}

	for { // Outer loop for attempting to subscribe and resubscribe
		select {
		case <-ctx.Done():
			log.Printf("Context done, exiting subscription goroutine permanently (reason: %v).", ctx.Err())
			return
		default:
			// Proceed to subscribe
		}

		log.Println("Attempting to subscribe to checkpoints...")
		stream, err := subClient.SubscribeCheckpoints(ctx, &subPb.SubscribeCheckpointsRequest{
			ReadMask: &fieldmaskpb.FieldMask{
				// We only require the aggregated signature (which includes
				// the epoch information) and the sequence number of the
				// checkpoint. Requesting fewer fields reduces payload size.
				Paths: []string{"signature", "sequence_number"},
			},
		})

		if err != nil {
			log.Printf("Failed to subscribe to checkpoints: %v", err)
			if ctx.Err() != nil {
				log.Println("Context cancelled during subscription attempt. Exiting.")
				return
			}
			log.Printf("Will retry subscription in %v...", retryDelay)
			select {
			case <-time.After(retryDelay):
				// Potentially increase retryDelay here for backoff
				continue
			case <-ctx.Done():
				log.Printf("Context done while waiting to retry subscription: %v. Exiting.", ctx.Err())
				return
			}
		}

		log.Println("Successfully subscribed. Waiting for checkpoints...")

	recvLoop:
		for {
			resp, err := stream.Recv() // resp is *subPb.SubscribeCheckpointsResponse
			if err != nil {
				if ctx.Err() != nil {
					log.Printf("Context cancelled during Recv(): %v. Exiting subscription.", ctx.Err())
					return
				}

				if err == io.EOF {
					log.Println("Checkpoint stream ended (EOF). Attempting to resubscribe...")
					break recvLoop
				}

				s, ok := status.FromError(err)
				if ok {
					if s.Code() == codes.Canceled {
						log.Printf("Subscription explicitly cancelled via gRPC context status (code: Canceled). Exiting: %v", err)
						return
					}
					if s.Code() == codes.Internal || s.Code() == codes.Unavailable { // Also handle Unavailable for network blips
						log.Printf("Stream terminated with gRPC error: %v (Code: %s). Attempting to resubscribe...", err, s.Code())
						break recvLoop
					}
					log.Printf("Unhandled gRPC error receiving checkpoint: %v (Code: %s). Attempting to resubscribe...", err, s.Code())
					break recvLoop
				} else {
					log.Printf("Non-gRPC error receiving checkpoint: %v. Attempting to resubscribe...", err)
					break recvLoop
				}
			}

			// Successfully received a response. resp.GetCheckpoint() is of type *subPb.CheckpointData
			if resp.GetCheckpoint() != nil {
				select {
				case checkpointChan <- resp.GetCheckpoint():
					// Successfully sent to channel
				case <-ctx.Done():
					log.Printf("Context done while trying to send checkpoint to channel: %v. Exiting.", ctx.Err())
					return
				}
			}
		} // End of recvLoop

		log.Printf("Disconnected from stream. Waiting %v before attempting to resubscribe...", retryDelay)
		select {
		case <-time.After(retryDelay):
			// Potentially increase retryDelay for exponential backoff if desired
		case <-ctx.Done():
			log.Printf("Context done while waiting to resubscribe after disconnection: %v. Exiting.", ctx.Err())
			return
		}
	} // End of outer subscription loop
}
