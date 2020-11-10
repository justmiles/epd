package cmd

import (
	"log"
	"strings"

	"github.com/justmiles/epd/lib/dashboard"
	"github.com/spf13/cobra"
)

func init() {
	log.SetFlags(0)
	rootCmd.AddCommand(displayTextCmd)
}

var displayTextCmd = &cobra.Command{
	Use:   "display-text",
	Short: "Display text on your EPD",
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) == 0 {
			errorOut("Please pass text to display")
		}

		d, err := dashboard.NewDashboard(device)
		if err != nil {
			panic(err)
		}

		if initialize {
			d.E.HardwareInit()
		}

		err = d.DisplayText(strings.Join(args, " "))
		if err != nil {
			panic(err)
		}

		if sleep {
			d.E.Sleep()
		}

	},
}
