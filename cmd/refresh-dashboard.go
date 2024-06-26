package cmd

import (
	"fmt"
	"log"

	dashboard "github.com/justmiles/epd/lib/dashboard"
	"github.com/spf13/cobra"
)

var (
	taskWarriorOptions dashboard.TaskWarriorOptions
	weatherAPIOptions  dashboard.WeatherAPIOptions
	dstaskOptions      dashboard.DstaskOptions
	location           string
)

func init() {
	rootCmd.AddCommand(refreshDashboardCmd)
	refreshDashboardCmd.PersistentFlags().BoolVar(&dstaskOptions.Enable, "dstask", false, "enable dstask")
	refreshDashboardCmd.PersistentFlags().BoolVar(&taskWarriorOptions.Enable, "taskwarrior", false, "enable taskwarrior")
	refreshDashboardCmd.PersistentFlags().StringVar(&taskWarriorOptions.ConfigPath, "taskwarrior-path", "", "path to your taskwarrior config")
	refreshDashboardCmd.PersistentFlags().StringVar(&weatherAPIOptions.WeatherAPIKey, "weather-api-key", "", "your openweathermap.org API key")
	refreshDashboardCmd.PersistentFlags().StringVar(&weatherAPIOptions.WeatherLanguage, "weather-language", "EN", "langage for weather")
	refreshDashboardCmd.PersistentFlags().StringVar(&weatherAPIOptions.WeatherTempUnit, "weather-temp-unit", "F", "temperature unit for weather")
	refreshDashboardCmd.PersistentFlags().StringVar(&weatherAPIOptions.WeatherCountry, "weather-country", "US", "temperature unit for weather")
	refreshDashboardCmd.PersistentFlags().StringVar(&location, "location", "America/New_York", "location for date")
	refreshDashboardCmd.PersistentFlags().IntVar(&weatherAPIOptions.WeatherZipCode, "weather-zip", 37069, "zip code for weather")
	refreshDashboardCmd.PersistentFlags().BoolVar(&previewImage, "preview", false, "preview the dashboard instead of updating the display")
}

var refreshDashboardCmd = &cobra.Command{
	Use:   "refresh-dashboard",
	Short: "Update your display with a custom dashboard",
	Run: func(cmd *cobra.Command, args []string) {

		d, err := dashboard.NewDashboard(
			dashboard.WithTaskWarrior(&taskWarriorOptions),
			dashboard.WithWeatherAPI(&weatherAPIOptions),
			dashboard.WithDstask(&dstaskOptions),
		)

		if err != nil {
			log.Fatalf("error creating custom dashboard: %s", err)
		}

		const outputImage = "dashboard-image.png"

		err = d.Generate(outputImage)
		if err != nil {
			log.Fatal(err)
		}

		if previewImage {
			return
		}
		// re-init with just the EPD so that we can generate local images when not connected to the EPD locally
		d, err = dashboard.NewDashboard(
			dashboard.WithEPD(device),
		)

		if initialize {
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("unable to connect to EPD")
				}
			}()
			d.EPDService.HardwareInit()
		}

		if err != nil {
			log.Fatal(err)
		}

		err = d.DisplayImage(outputImage)
		if err != nil {
			log.Fatal(err)
		}

		if sleep {
			d.EPDService.Sleep()
		}

	},
}
