package cmd

import (
	"fmt"
	"log"
	"os"

	epd7in5v2 "github.com/justmiles/epd/lib/epd7in5v2"

	"github.com/spf13/cobra"
)

var (
	debug  bool
	device string
)

func init() {
	log.SetFlags(0)
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")
	rootCmd.PersistentFlags().StringVarP(&device, "device", "d", "epd7in5v2", "select device type")
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "epd",
	Short: "Interact with a supported Electronic Paper Display",
	Run: func(rootCmd *cobra.Command, args []string) {
		epd, err := epd7in5v2.NewRaspberryPiHat()
		if err != nil {
			panic(err)
		}

		err = epd.DisplayText(`Fuck it!`)
		if err != nil {
			panic(err)
		}

	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(version string) {
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(`{{with .Name}}{{printf "%s " .}}{{end}}{{printf "%s" .Version}}
`)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func newEPD() (e *epd7in5v2.EPD, err error) {
	e, err = epd7in5v2.NewRaspberryPiHat()
	if err != nil {
		return nil, err
	}

	if device != "epd7in5v2" {
		return nil, fmt.Errorf("Device %s is not supported", device)
	}

	return e, nil
}
