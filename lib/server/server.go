package server

import (
	"context"
	"fmt"
	"log"

	"github.com/justmiles/epd/lib/display"
	pb "github.com/justmiles/epd/proto/epdpb"
)

// EPDServer implements the gRPC EPDService server interface.
type EPDServer struct {
	pb.UnimplementedEPDServiceServer
	display *display.LocalDisplay
}

// NewEPDServer creates a new gRPC server backed by a local display.
func NewEPDServer(device string) (*EPDServer, error) {
	d, err := display.NewLocalDisplay(device)
	if err != nil {
		return nil, fmt.Errorf("failed to create local display: %w", err)
	}

	// Initialize hardware on startup
	d.HardwareInit()
	log.Printf("EPD hardware initialized (device: %s)", device)

	return &EPDServer{
		display: d,
	}, nil
}

// DisplayImage receives PNG data and displays it on the EPD.
func (s *EPDServer) DisplayImage(ctx context.Context, req *pb.DisplayImageRequest) (*pb.DisplayImageResponse, error) {
	log.Printf("Received DisplayImage request (%d bytes)", len(req.ImageData))

	if err := s.display.DisplayImage(req.ImageData); err != nil {
		log.Printf("DisplayImage error: %v", err)
		return nil, fmt.Errorf("failed to display image: %w", err)
	}

	log.Println("Image displayed successfully")
	return &pb.DisplayImageResponse{Message: "Image displayed successfully"}, nil
}

// DisplayText renders text and displays it on the EPD.
func (s *EPDServer) DisplayText(ctx context.Context, req *pb.DisplayTextRequest) (*pb.DisplayTextResponse, error) {
	log.Printf("Received DisplayText request: %q", req.Text)

	if err := s.display.DisplayText(req.Text); err != nil {
		log.Printf("DisplayText error: %v", err)
		return nil, fmt.Errorf("failed to display text: %w", err)
	}

	log.Println("Text displayed successfully")
	return &pb.DisplayTextResponse{Message: "Text displayed successfully"}, nil
}

// Clear clears the EPD to white.
func (s *EPDServer) Clear(ctx context.Context, req *pb.ClearRequest) (*pb.ClearResponse, error) {
	log.Println("Received Clear request")

	if err := s.display.Clear(); err != nil {
		log.Printf("Clear error: %v", err)
		return nil, fmt.Errorf("failed to clear display: %w", err)
	}

	log.Println("Display cleared successfully")
	return &pb.ClearResponse{Message: "Display cleared"}, nil
}

// Sleep puts the EPD into sleep mode.
func (s *EPDServer) Sleep(ctx context.Context, req *pb.SleepRequest) (*pb.SleepResponse, error) {
	log.Println("Received Sleep request")

	if err := s.display.Sleep(); err != nil {
		log.Printf("Sleep error: %v", err)
		return nil, fmt.Errorf("failed to sleep display: %w", err)
	}

	log.Println("Display set to sleep mode")
	return &pb.SleepResponse{Message: "Display sleeping"}, nil
}

// Shutdown gracefully shuts down the server, putting the display to sleep.
func (s *EPDServer) Shutdown() {
	log.Println("Shutting down EPD server...")
	if err := s.display.Sleep(); err != nil {
		log.Printf("Warning: failed to sleep display on shutdown: %v", err)
	}
	s.display.Close()
}
