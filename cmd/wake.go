package cmd

import (
	"log"

	"github.com/justmiles/epd/lib/dashboard"
	"github.com/spf13/cobra"
)

func init() {
	log.SetFlags(0)
	rootCmd.AddCommand(wakeCmd)
}

var wakeCmd = &cobra.Command{
	Use:   "wake",
	Short: "Intialize the EPD. Required when coming out of sleep mode or rebooting",
	Run: func(cmd *cobra.Command, args []string) {

		d, err := dashboard.NewDashboard(device)
		if err != nil {
			panic(err)
		}

		d.E.HardwareInit()

	},
}
