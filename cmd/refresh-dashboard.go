package cmd

import (
	"fmt"
	"log"
	"os"

	dashboard "github.com/justmiles/epd/lib/dashboard"
	"github.com/justmiles/epd/lib/display"
	"github.com/spf13/cobra"
)

var (
	weatherAPIOptions dashboard.WeatherAPIOptions
	location          string
	headerText        string
	bodyText          string
)

func init() {
	rootCmd.AddCommand(refreshDashboardCmd)
	refreshDashboardCmd.PersistentFlags().StringVar(&weatherAPIOptions.WeatherAPIKey, "weather-api-key", envDefault("EPD_WEATHER_API_KEY", ""), "your openweathermap.org API key (env: EPD_WEATHER_API_KEY)")
	refreshDashboardCmd.PersistentFlags().StringVar(&weatherAPIOptions.WeatherLanguage, "weather-language", envDefault("EPD_WEATHER_LANGUAGE", "EN"), "language for weather (env: EPD_WEATHER_LANGUAGE)")
	refreshDashboardCmd.PersistentFlags().StringVar(&weatherAPIOptions.WeatherTempUnit, "weather-temp-unit", envDefault("EPD_WEATHER_TEMP_UNIT", "F"), "temperature unit for weather (env: EPD_WEATHER_TEMP_UNIT)")
	refreshDashboardCmd.PersistentFlags().StringVar(&weatherAPIOptions.WeatherCountry, "weather-country", envDefault("EPD_WEATHER_COUNTRY", "US"), "country for weather (env: EPD_WEATHER_COUNTRY)")
	refreshDashboardCmd.PersistentFlags().StringVar(&location, "location", envDefault("EPD_LOCATION", "America/Chicago"), "location for date (env: EPD_LOCATION)")
	refreshDashboardCmd.PersistentFlags().IntVar(&weatherAPIOptions.WeatherZipCode, "weather-zip", envDefaultInt("EPD_WEATHER_ZIP", 60601), "zip code for weather (env: EPD_WEATHER_ZIP)")
	refreshDashboardCmd.PersistentFlags().StringVar(&headerText, "header-text", envDefault("EPD_HEADER_TEXT", ""), "custom header text for the dashboard (env: EPD_HEADER_TEXT)")
	refreshDashboardCmd.PersistentFlags().StringVar(&bodyText, "body-text", envDefault("EPD_BODY_TEXT", ""), "custom body text for the dashboard (env: EPD_BODY_TEXT)")
	refreshDashboardCmd.PersistentFlags().BoolVar(&previewImage, "preview", false, "preview the dashboard instead of updating the display")
}

var refreshDashboardCmd = &cobra.Command{
	Use:   "refresh-dashboard",
	Short: "Update your display with a custom dashboard",
	Run: func(cmd *cobra.Command, args []string) {

		// Generate the dashboard image locally (no EPD needed for generation)
		d, err := dashboard.NewDashboard(
			dashboard.WithWeatherAPI(&weatherAPIOptions),
		)

		if err != nil {
			log.Fatalf("error creating custom dashboard: %s", err)
		}

		const outputImage = "dashboard-image.png"

		// If headerText or bodyText point to a file, read the file contents
		headerText = resolveTextOrFile(headerText)
		bodyText = resolveTextOrFile(bodyText)

		err = d.Generate(outputImage, headerText, bodyText)
		if err != nil {
			log.Fatal(err)
		}

		if previewImage {
			return
		}

		// Now push the generated image to the display (local or remote)
		svc, err := newDisplayService(device, initialize)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer svc.Close()

		if display.IsRemote(device) {
			// Read the generated PNG and send via gRPC
			pngData, err := os.ReadFile(outputImage)
			if err != nil {
				log.Fatalf("error reading generated dashboard: %v", err)
			}
			if err := svc.DisplayImage(pngData); err != nil {
				log.Fatal(err)
			}
		} else {
			// Local display: use direct file path
			local := svc.(*display.LocalDisplay)
			if err := local.DisplayImageFromFile(outputImage); err != nil {
				log.Fatal(err)
			}
		}

		if sleep {
			svc.Sleep()
		}
	},
}
