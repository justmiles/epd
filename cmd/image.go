package cmd

import (
	"log"

	"github.com/justmiles/epd/lib/dashboard"
	"github.com/spf13/cobra"
)

var (
	previewImage bool
)

func init() {
	log.SetFlags(0)
	rootCmd.AddCommand(displayImageCmd)

	displayImageCmd.PersistentFlags().BoolVar(&previewImage, "preview", false, "preview the image instead of updating the display")
}

var displayImageCmd = &cobra.Command{
	Use:   "display-image",
	Short: "Display an image on your EPD",
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) == 0 {
			errorOut("Please pass an image to display")
		}

		displayImage := args[0]

		d, err := dashboard.NewDashboard(dashboard.WithEPD(device))
		if err != nil {
			errorOut(err.Error())
		}

		if initialize {
			d.EPDService.HardwareInit()
		}

		err = d.DisplayImage(displayImage)
		if err != nil {
			errorOut(err.Error())
		}

		if sleep {
			d.EPDService.Sleep()
		}

	},
}
