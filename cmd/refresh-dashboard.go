package cmd

import (
	"log"

	dashboard "github.com/justmiles/epd/lib/dashboard"
	"github.com/spf13/cobra"
)

var (
	taskWarriorOptions dashboard.TaskWarriorOptions
	weatherAPIOptions  dashboard.WeatherAPIOptions
)

func init() {
	rootCmd.AddCommand(refreshDashboardCmd)
	refreshDashboardCmd.PersistentFlags().StringVar(&taskWarriorOptions.ConfigPath, "taskwarrior-path", "", "path to your taskwarrior config")
	refreshDashboardCmd.PersistentFlags().StringVar(&weatherAPIOptions.WeatherAPIKey, "weather-api-key", "", "your openweathermap.org API key")
	refreshDashboardCmd.PersistentFlags().StringVar(&weatherAPIOptions.WeatherLanguage, "weather-language", "EN", "langage for weather")
	refreshDashboardCmd.PersistentFlags().StringVar(&weatherAPIOptions.WeatherTempUnit, "weather-temp-unit", "F", "temperature unit for weather")
	refreshDashboardCmd.PersistentFlags().StringVar(&weatherAPIOptions.WeatherCountry, "weather-country", "US", "temperature unit for weather")
	refreshDashboardCmd.PersistentFlags().IntVar(&weatherAPIOptions.WeatherZipCode, "weather-zip", 37069, "zip code for weather")
}

var refreshDashboardCmd = &cobra.Command{
	Use:   "refresh-dashboard",
	Short: "",
	Run: func(cmd *cobra.Command, args []string) {

		dashboard, err := dashboard.NewDashboard(
			dashboard.WithEPD(device),
			dashboard.WithTaskWarrior(&taskWarriorOptions),
			dashboard.WithWeatherAPI(&weatherAPIOptions),
		)

		if err != nil {
			log.Fatalf("error creating custom dashboard: %s", err)
		}

		const outputImage = "dashboard-image.png"

		err = dashboard.Generate(outputImage)
		if err != nil {
			log.Fatal(err)
		}

		err = dashboard.DisplayImage(outputImage)
		if err != nil {
			log.Fatal(err)
		}

	},
}
