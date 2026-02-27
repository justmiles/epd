package display

import (
	"context"
	"fmt"
	"time"

	pb "github.com/justmiles/epd/proto/epdpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// RemoteDisplay implements Service by forwarding calls to a remote daemon via gRPC.
type RemoteDisplay struct {
	conn   *grpc.ClientConn
	client pb.EPDServiceClient
	addr   string
}

// NewRemoteDisplay creates a new remote display service connected to the given address.
func NewRemoteDisplay(address string) (*RemoteDisplay, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to EPD daemon at %s: %w", address, err)
	}

	return &RemoteDisplay{
		conn:   conn,
		client: pb.NewEPDServiceClient(conn),
		addr:   address,
	}, nil
}

// DisplayImage sends raw PNG data to the remote daemon for display.
func (r *RemoteDisplay) DisplayImage(pngData []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_, err := r.client.DisplayImage(ctx, &pb.DisplayImageRequest{
		ImageData: pngData,
	})
	if err != nil {
		return fmt.Errorf("remote DisplayImage failed: %w", err)
	}
	return nil
}

// DisplayText sends text to the remote daemon for rendering and display.
func (r *RemoteDisplay) DisplayText(text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_, err := r.client.DisplayText(ctx, &pb.DisplayTextRequest{
		Text: text,
	})
	if err != nil {
		return fmt.Errorf("remote DisplayText failed: %w", err)
	}
	return nil
}

// Clear sends a clear command to the remote daemon.
func (r *RemoteDisplay) Clear() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := r.client.Clear(ctx, &pb.ClearRequest{})
	if err != nil {
		return fmt.Errorf("remote Clear failed: %w", err)
	}
	return nil
}

// Sleep sends a sleep command to the remote daemon.
func (r *RemoteDisplay) Sleep() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := r.client.Sleep(ctx, &pb.SleepRequest{})
	if err != nil {
		return fmt.Errorf("remote Sleep failed: %w", err)
	}
	return nil
}

// Close closes the gRPC connection.
func (r *RemoteDisplay) Close() error {
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}
