package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/justmiles/epd/lib/display"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(clearCmd)
}

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the EPD to white",
	Run: func(cmd *cobra.Command, args []string) {

		svc, err := newDisplayService(device, initialize)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer svc.Close()

		if err := svc.Clear(); err != nil {
			log.Fatal(err)
		}

		if sleep {
			svc.Sleep()
		}
	},
}

// newDisplayService creates a local or remote DisplayService based on the device string.
func newDisplayService(dev string, init bool) (display.Service, error) {
	if display.IsRemote(dev) {
		return display.NewRemoteDisplay(dev)
	}

	local, err := display.NewLocalDisplay(dev)
	if err != nil {
		return nil, err
	}

	if init {
		local.HardwareInit()
	}

	return local, nil
}
