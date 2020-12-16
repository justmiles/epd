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

		cd, err := dashboard.NewDashboard(dashboard.WithEPD(device))
		if err != nil {
			errorOut(err.Error())
		}

		if initialize {
			cd.EPDService.HardwareInit()
		}

		err = cd.DisplayText(strings.Join(args, " "))
		if err != nil {
			panic(err)
		}

		if sleep {
			cd.EPDService.Sleep()
		}

	},
}
