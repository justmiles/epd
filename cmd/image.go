package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/justmiles/epd/lib/display"
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

		imagePath := args[0]

		svc, err := newDisplayService(device, initialize)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer svc.Close()

		// For remote display, read and encode the image locally then send PNG bytes
		if display.IsRemote(device) {
			pngData, err := display.ReadImageFile(imagePath, 800, 480)
			if err != nil {
				errorOut(err.Error())
			}
			if err := svc.DisplayImage(pngData); err != nil {
				errorOut(err.Error())
			}
		} else {
			// For local display, use the direct file path method
			local := svc.(*display.LocalDisplay)
			if err := local.DisplayImageFromFile(imagePath); err != nil {
				errorOut(err.Error())
			}
		}

		if sleep {
			svc.Sleep()
		}
	},
}
