package cmd

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/justmiles/epd/lib/display"
	"github.com/justmiles/epd/lib/server"
	pb "github.com/justmiles/epd/proto/epdpb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var (
	servePort                int
	refreshDashboardInterval time.Duration
)

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.PersistentFlags().IntVar(&servePort, "port", 50051, "gRPC server port")
	serveCmd.PersistentFlags().DurationVar(&refreshDashboardInterval, "refresh-dashboard-interval", 0,
		"automatically refresh the dashboard at this interval (e.g. 5m, 1h); requires Tududi or --body-text flags")
	// Register dashboard content flags so the scheduled refresh can be configured
	// via the same flags/env-vars as refresh-dashboard.
	registerDashboardRefreshFlags(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run as a daemon, exposing the EPD over gRPC",
	Long: `Run the EPD as a daemon process that listens for gRPC requests.
Other machines can then use the display-image and display-text commands
with --device host:port to push content to this display remotely.

Pass --refresh-dashboard-interval to also refresh the dashboard on a schedule:

  epd serve --refresh-dashboard-interval 5m \
            --tududi-url http://localhost:3004 \
            --tududi-api-token $TOKEN`,
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

		// Scheduled dashboard refresh — connects back via loopback gRPC so that
		// all display access is serialized through the server.
		if refreshDashboardInterval > 0 {
			go func() {
				localAddr := fmt.Sprintf("localhost:%d", servePort)
				log.Printf("Dashboard refresh enabled every %s (connecting to %s)", refreshDashboardInterval, localAddr)

				// Retry connecting until the gRPC server is ready
				var svc *display.RemoteDisplay
				for attempts := 0; attempts < 10; attempts++ {
					time.Sleep(500 * time.Millisecond)
					svc, err = display.NewRemoteDisplay(localAddr)
					if err == nil {
						break
					}
					log.Printf("Dashboard refresh: waiting for gRPC server... (%v)", err)
				}
				if svc == nil {
					log.Printf("Dashboard refresh: could not connect to local gRPC server, skipping")
					return
				}
				defer svc.Close()

				do := func() {
					log.Printf("Refreshing dashboard...")
					if err := runDashboardRefresh(svc); err != nil {
						log.Printf("Dashboard refresh error: %v", err)
					}
				}

				do() // run once immediately on start
				ticker := time.NewTicker(refreshDashboardInterval)
				defer ticker.Stop()
				for range ticker.C {
					do()
				}
			}()
		}

		log.Printf("EPD daemon listening on :%d (device: %s)", servePort, device)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	},
}
