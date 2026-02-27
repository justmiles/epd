package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

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

		svc, err := newDisplayService(device, initialize)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer svc.Close()

		if err := svc.DisplayText(strings.Join(args, " ")); err != nil {
			errorOut(err.Error())
		}

		if sleep {
			svc.Sleep()
		}
	},
}
