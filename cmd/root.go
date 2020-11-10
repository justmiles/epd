package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var (
	debug, initialize, sleep bool
	device                   string
)

func init() {
	log.SetFlags(0)
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")
	rootCmd.PersistentFlags().StringVarP(&device, "device", "d", "epd7in5v2", "select device type")
	rootCmd.PersistentFlags().BoolVarP(&initialize, "initialize", "i", false, "initialize (wake) the device before updating it. Required if in sleep mode")
	rootCmd.PersistentFlags().BoolVarP(&sleep, "sleep", "s", false, "set the device to sleep mode after updating display")
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "epd",
	Short: "Interact with a supported Electronic Paper Display",
	Run: func(rootCmd *cobra.Command, args []string) {
		// TODO display help
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

func errorOut(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}
