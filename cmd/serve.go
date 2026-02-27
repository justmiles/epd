package cmd

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/justmiles/epd/lib/server"
	pb "github.com/justmiles/epd/proto/epdpb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var servePort int

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.PersistentFlags().IntVar(&servePort, "port", 50051, "gRPC server port")
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run as a daemon, exposing the EPD over gRPC",
	Long: `Run the EPD as a daemon process that listens for gRPC requests.
Other machines can then use the display-image and display-text commands
with --device host:port to push content to this display remotely.`,
	Run: func(cmd *cobra.Command, args []string) {

		// Use the root --device flag for the local hardware device type
		epdServer, err := server.NewEPDServer(device)
		if err != nil {
			log.Fatalf("Failed to initialize EPD server: %v", err)
		}

		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", servePort))
		if err != nil {
			log.Fatalf("Failed to listen on port %d: %v", servePort, err)
		}

		grpcServer := grpc.NewServer()
		pb.RegisterEPDServiceServer(grpcServer, epdServer)

		// Graceful shutdown
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			log.Println("Received shutdown signal")
			epdServer.Shutdown()
			grpcServer.GracefulStop()
		}()

		log.Printf("EPD daemon listening on :%d (device: %s)", servePort, device)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	},
}
