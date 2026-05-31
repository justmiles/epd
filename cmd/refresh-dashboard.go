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
	registerDashboardRefreshFlags(refreshDashboardCmd)
	refreshDashboardCmd.PersistentFlags().BoolVar(&previewImage, "preview", false, "preview the dashboard instead of updating the display")
}

// registerDashboardRefreshFlags registers all dashboard/content flags onto cmd.
// Called by both refresh-dashboard and serve (for the scheduled refresh feature).
func registerDashboardRefreshFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&weatherAPIOptions.WeatherAPIKey, "weather-api-key", envDefault("EPD_WEATHER_API_KEY", ""), "your openweathermap.org API key (env: EPD_WEATHER_API_KEY)")
	cmd.PersistentFlags().StringVar(&weatherAPIOptions.WeatherLanguage, "weather-language", envDefault("EPD_WEATHER_LANGUAGE", "EN"), "language for weather (env: EPD_WEATHER_LANGUAGE)")
	cmd.PersistentFlags().StringVar(&weatherAPIOptions.WeatherTempUnit, "weather-temp-unit", envDefault("EPD_WEATHER_TEMP_UNIT", "F"), "temperature unit for weather (env: EPD_WEATHER_TEMP_UNIT)")
	cmd.PersistentFlags().StringVar(&weatherAPIOptions.WeatherCountry, "weather-country", envDefault("EPD_WEATHER_COUNTRY", "US"), "country for weather (env: EPD_WEATHER_COUNTRY)")
	cmd.PersistentFlags().StringVar(&location, "location", envDefault("EPD_LOCATION", "America/Chicago"), "location for date (env: EPD_LOCATION)")
	cmd.PersistentFlags().IntVar(&weatherAPIOptions.WeatherZipCode, "weather-zip", envDefaultInt("EPD_WEATHER_ZIP", 60601), "zip code for weather (env: EPD_WEATHER_ZIP)")
	cmd.PersistentFlags().StringVar(&headerText, "header-text", envDefault("EPD_HEADER_TEXT", ""), "custom header text for the dashboard (env: EPD_HEADER_TEXT)")
	cmd.PersistentFlags().StringVar(&bodyText, "body-text", envDefault("EPD_BODY_TEXT", ""), "custom body text for the dashboard (env: EPD_BODY_TEXT)")
	cmd.PersistentFlags().StringVar(&tududiOptions.URL, "tududi-url", envDefault("TUDUDI_URL", ""), "Tududi base URL, e.g. http://localhost:3004 (env: TUDUDI_URL)")
	cmd.PersistentFlags().StringVar(&tududiOptions.APIToken, "tududi-api-token", envDefault("TUDUDI_API_TOKEN", ""), "Tududi API bearer token (env: TUDUDI_API_TOKEN)")
	cmd.PersistentFlags().StringVar(&tududiOptions.Project, "tududi-project", envDefault("TUDUDI_PROJECT", ""), "only show tasks in this project — ID or name (env: TUDUDI_PROJECT)")
	cmd.PersistentFlags().IntVar(&tududiOptions.MaxTasks, "tududi-max-tasks", envDefaultInt("TUDUDI_MAX_TASKS", 15), "max number of Tududi tasks to display (env: TUDUDI_MAX_TASKS)")
}

var refreshDashboardCmd = &cobra.Command{
	Use:   "refresh-dashboard",
	Short: "Update your display with a custom dashboard",
	Run: func(cmd *cobra.Command, args []string) {
		const outputImage = "dashboard-image.png"

		if err := generateDashboardPNG(outputImage); err != nil {
			log.Fatal(err)
		}

		if previewImage {
			return
		}

		svc, err := newDisplayService(device, initialize)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer svc.Close()

		if display.IsRemote(device) {
			pngData, err := os.ReadFile(outputImage)
			if err != nil {
				log.Fatalf("error reading generated dashboard: %v", err)
			}
			if err := svc.DisplayImage(pngData); err != nil {
				log.Fatal(err)
			}
		} else {
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

// generateDashboardPNG resolves content (header/body text, Tududi) and writes
// the finished dashboard PNG to outputFile.
func generateDashboardPNG(outputFile string) error {
	d, err := dashboard.NewDashboard(
		dashboard.WithWeatherAPI(&weatherAPIOptions),
	)
	if err != nil {
		return fmt.Errorf("creating dashboard: %w", err)
	}

	hdr := resolveTextOrFile(headerText)
	body := resolveTextOrFile(bodyText)

	// Tududi: only used when --body-text is not set
	if body == "" && tududiOptions.URL != "" && tududiOptions.APIToken != "" {
		var autoHeader string
		body, autoHeader, err = fetchTududiBodyText(tududiOptions)
		if err != nil {
			return fmt.Errorf("fetching Tududi tasks: %w", err)
		}
		if hdr == "" && autoHeader != "" {
			hdr = autoHeader
		}
		if hdr == "" {
			hdr = "Tududi Tasks"
		}
	}

	return d.Generate(outputFile, hdr, body)
}

// runDashboardRefresh generates the dashboard PNG and pushes it to svc.
// Used by both the refresh-dashboard command and the serve periodic refresh.
func runDashboardRefresh(svc display.Service) error {
	const outputImage = "dashboard-image.png"
	if err := generateDashboardPNG(outputImage); err != nil {
		return err
	}
	pngData, err := os.ReadFile(outputImage)
	if err != nil {
		return fmt.Errorf("reading dashboard image: %w", err)
	}
	return svc.DisplayImage(pngData)
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
