package dashboard

import (
	"fmt"
	"image"
	"math"
	"time"

	"github.com/fogleman/gg"
)

// WeatherAPIOptions defines options for the WeatherAPI
type WeatherAPIOptions struct {
	WeatherAPIKey   string
	WeatherLanguage string
	WeatherTempUnit string
	WeatherCountry  string
	WeatherZipCode  int
}

// WithWeatherAPI creates a custom dashboard with Weather API configured
func WithWeatherAPI(weatherAPIOptions *WeatherAPIOptions) Options {
	return func(d *Dashboard) {
		d.weatherAPIOptions = weatherAPIOptions
	}
}

// buildWeatherWidget x pixels wide and y pixels tall
func (d *Dashboard) buildWeatherWidget(x, y int) (image.Image, error) {
	var (
		xWidth, xHeight = float64(x), float64(y)
	)

	// Draw background
	dc := gg.NewContext(x, y)
	dc.DrawRectangle(0, 0, xWidth, xHeight)
	dc.SetRGB(0, 0, 0)
	dc.Fill()

	// Draw HR
	dc.DrawRectangle(0+10, 0, xWidth-20, 2)
	dc.SetRGB(1, 1, 1)
	dc.Fill()

	// Get Weather info
	d.weatherAPIService.CurrentByZip(d.weatherAPIOptions.WeatherZipCode, d.weatherAPIOptions.WeatherCountry)
	if len(d.weatherAPIService.Weather) == 0 {
		return nil, fmt.Errorf("Unable to get weather data")
	}

	drawCurrent(dc, d.weatherAPIService.Weather[0].Icon, d.weatherAPIService.Weather[0].Main, d.weatherAPIService.Main.Temp)

	// Draw Hourly
	// Using GeoPos which is correclty seeded from CurrentByZip() above
	err := d.weatherAPIOneCall.OneCallByCoordinates(&d.weatherAPIService.GeoPos)
	if err != nil {
		return nil, fmt.Errorf("Unable to get hourly weather data: %s", err)
	}

	hourlyX := 15
	for i, val := range d.weatherAPIOneCall.Hourly {
		if i == 0 {
			continue
		}
		if i >= 10 {
			break
		}
		fmt.Printf("Time: %v Temp: %v\n", val.Dt, val.Temp)

		drawHourly(dc, hourlyX, 120, d.weatherAPIService.Weather[0].Icon, "", val.Temp, val.Dt)
		hourlyX += 18 + 5
	}

	return dc.Image(), nil
}

func drawCurrent(dc *gg.Context, icon string, description string, temp float64) {
	// Icon
	img := convertSVGToImage(getIcon(icon), 64, 64)
	dc.DrawImageAnchored(img, 32, 32, 0, 0)

	// Description
	dc.SetRGB(1, 1, 1)
	setDynamicFont(dc, 128, 32, description)
	dc.DrawStringAnchored(description, 170, 32, 0.5, 0.5)

	// Temp
	t := fmt.Sprintf("%v°", math.Floor(temp))
	setFont(dc, 32)
	dc.DrawStringAnchored(t, 170, 74, 0.5, 0.5)
}

func drawHourly(dc *gg.Context, x int, y int, icon string, description string, temp float64, dt int) {
	width := float64(18)

	// Icon
	img := convertSVGToImage(getIcon(icon), 15, 15)
	dc.DrawImageAnchored(img, x+2, y, 0, 0)

	// Draw HR
	dc.DrawRectangle(float64(x), float64(y)+20, width, 2)
	dc.SetRGB(1, 1, 1)
	dc.Fill()

	// Temp
	t := fmt.Sprintf("%v°", math.Floor(temp))
	setFont(dc, 12)
	dc.DrawStringAnchored(t, float64(x)+(width/2), float64(y), 0.4, 3)

	// Hour
	loc, _ := time.LoadLocation("America/New_York")

	hour := fmt.Sprintf("%v", time.Unix(int64(dt), 0).In(loc).Hour())

	dc.DrawStringAnchored(hour, float64(x)+(width/2), float64(y), 0.4, 4)
}
