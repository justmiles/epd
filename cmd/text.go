package cmd

import (
	"log"

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

		e, err := newEPD()
		if err != nil {
			panic(err)
		}

		err = e.DisplayText(displayText)
		if err != nil {
			panic(err)
		}

	},
}
