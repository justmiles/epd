package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var (
	displayImage string
)

func init() {
	log.SetFlags(0)
	rootCmd.AddCommand(displayImageCmd)

	displayImageCmd.PersistentFlags().StringVarP(&displayImage, "image", "i", "", "image to display")
}

var displayImageCmd = &cobra.Command{
	Use:   "display-image",
	Short: "Display an image on your EPD",
	Run: func(cmd *cobra.Command, args []string) {

		e, err := newEPD()
		if err != nil {
			panic(err)
		}

		err = e.DisplayImage(displayImage)
		if err != nil {
			panic(err)
		}

	},
}
