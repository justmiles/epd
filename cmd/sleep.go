package cmd

import (
	"log"

	"github.com/justmiles/epd/lib/dashboard"
	"github.com/spf13/cobra"
)

func init() {
	log.SetFlags(0)
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "sleep",
	Short: "Set the EPD into sleep mode reducing power consumption",
	Run: func(cmd *cobra.Command, args []string) {

		d, err := dashboard.NewDashboard(device)
		if err != nil {
			panic(err)
		}

		d.E.Sleep()

	},
}
