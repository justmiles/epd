package cmd

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	debug, initialize, sleep bool
	device                   string
)

func init() {
	log.SetFlags(0)
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")
	rootCmd.PersistentFlags().StringVarP(&device, "device", "d", envDefault("EPD_DEVICE", "epd7in5v2"), "your supported EPD device type or remote host:port (env: EPD_DEVICE)")
	rootCmd.PersistentFlags().BoolVarP(&initialize, "initialize", "i", false, "initialize (wake) the device before updating it. Required if in sleep mode")
	rootCmd.PersistentFlags().BoolVarP(&sleep, "sleep", "s", false, "set the device to sleep mode after updating display")
}

// envDefault returns the value of the environment variable if set, otherwise the fallback.
func envDefault(envVar, fallback string) string {
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	return fallback
}

// envDefaultInt returns the integer value of the environment variable if set, otherwise the fallback.
func envDefaultInt(envVar string, fallback int) int {
	if v := os.Getenv(envVar); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

// resolveTextOrFile checks if the input string is a path to an existing file.
// If so, it reads and returns the file contents. Otherwise, it returns the string as-is.
func resolveTextOrFile(s string) string {
	if s == "" {
		return s
	}
	if info, err := os.Stat(s); err == nil && !info.IsDir() {
		data, err := os.ReadFile(s)
		if err == nil {
			return string(data)
		}
	}
	return s
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "epd",
	Short: "Interact with a supported Electronic Paper Display",
	Run: func(rootCmd *cobra.Command, args []string) {
		// TODO display halp
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
