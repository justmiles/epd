package cmd

import (
	"log"

	"github.com/justmiles/epd/lib/dashboard"
	"github.com/spf13/cobra"
)

var (
	displayText string
)

func init() {
	log.SetFlags(0)
	rootCmd.AddCommand(displayTextCmd)

	displayTextCmd.PersistentFlags().StringVarP(&displayText, "text", "t", "", "text to display")
}

var displayTextCmd = &cobra.Command{
	Use:   "display-text",
	Short: "Display text on your EPD",
	Run: func(cmd *cobra.Command, args []string) {

		d, err := dashboard.NewDashboard(device)
		if err != nil {
			panic(err)
		}

		err = d.DisplayText(displayText)
		if err != nil {
			panic(err)
		}

	},
}
