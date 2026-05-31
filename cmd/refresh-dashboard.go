package cmd

import (
	"fmt"
	"log"
	"os"
	"strconv"

	dashboard "github.com/justmiles/epd/lib/dashboard"
	"github.com/justmiles/epd/lib/display"
	"github.com/justmiles/epd/lib/tududi"
	"github.com/spf13/cobra"
)

var (
	weatherAPIOptions dashboard.WeatherAPIOptions
	location          string
	headerText        string
	bodyText          string
	tududiOptions     tududi.Options
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

	// Tududi provider flags
	refreshDashboardCmd.PersistentFlags().StringVar(&tududiOptions.URL, "tududi-url", envDefault("TUDUDI_URL", ""), "Tududi base URL, e.g. http://localhost:3004 (env: TUDUDI_URL)")
	refreshDashboardCmd.PersistentFlags().StringVar(&tududiOptions.APIToken, "tududi-api-token", envDefault("TUDUDI_API_TOKEN", ""), "Tududi API bearer token (env: TUDUDI_API_TOKEN)")
	refreshDashboardCmd.PersistentFlags().StringVar(&tududiOptions.Project, "tududi-project", envDefault("TUDUDI_PROJECT", ""), "only show tasks in this project — ID or name (env: TUDUDI_PROJECT)")
	refreshDashboardCmd.PersistentFlags().IntVar(&tududiOptions.MaxTasks, "tududi-max-tasks", envDefaultInt("TUDUDI_MAX_TASKS", 15), "max number of Tududi tasks to display (env: TUDUDI_MAX_TASKS)")
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

		// If --body-text is not set, attempt to populate it from Tududi.
		// --body-text always takes precedence over Tududi.
		if bodyText == "" && tududiOptions.URL != "" && tududiOptions.APIToken != "" {
			var autoHeader string
			bodyText, autoHeader, err = fetchTududiBodyText(tududiOptions)
			if err != nil {
				log.Fatalf("error fetching tasks from Tududi: %s", err)
			}
			// Use the resolved project name as the header when --header-text wasn't provided
			if headerText == "" && autoHeader != "" {
				headerText = autoHeader
			}
		}

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

// fetchTududiBodyText connects to the Tududi API and returns:
//   - body: a GFM Markdown string ready to pass to dashboard.Generate()
//   - autoHeader: the resolved project name when filtering by project (empty otherwise)
func fetchTududiBodyText(opts tududi.Options) (body, autoHeader string, err error) {
	client := tududi.NewClient(opts.URL, opts.APIToken)

	var projectID *int

	if opts.Project != "" {
		// Try integer ID first, then fall back to name match
		if id, e := strconv.Atoi(opts.Project); e == nil {
			projectID = &id
			// Resolve the name for the auto-header
			proj, e2 := client.FindProjectByName(opts.Project)
			if e2 == nil {
				autoHeader = proj.Name
			}
		} else {
			proj, e2 := client.FindProjectByName(opts.Project)
			if e2 != nil {
				return "", "", fmt.Errorf("finding project %q: %w", opts.Project, e2)
			}
			projectID = &proj.ID
			autoHeader = proj.Name
		}
	}

	tasks, err := client.ListTasks(projectID)
	if err != nil {
		return "", "", fmt.Errorf("listing tasks: %w", err)
	}

	if projectID != nil {
		// Single-project mode: flat list
		body = tududi.FormatTasksAsMarkdown(tasks, opts.MaxTasks)
	} else {
		// All-projects mode: grouped by project, top 3 per group
		maxPerProject := opts.MaxTasks
		if maxPerProject <= 0 || maxPerProject > 3 {
			maxPerProject = 3
		}
		projects, e := client.ListProjects()
		projectNames := make(map[int]string)
		if e == nil {
			for _, p := range projects {
				projectNames[p.ID] = p.Name
			}
		}
		body = tududi.FormatTasksByProject(tasks, projectNames, maxPerProject)
	}

	return body, autoHeader, nil
}

