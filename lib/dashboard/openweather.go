package dashboard

import (
	"fmt"
	"image"
	"math"

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

	img := convertSVGToImage(getIcon(d.weatherAPIService.Weather[0].Icon), 64, 64)
	dc.DrawImageAnchored(img, 32, 32, 0, 0)

	// Draw weather description
	dc.SetRGB(1, 1, 1)
	setDynamicFont(dc, 128, 32, d.weatherAPIService.Weather[0].Main)
	dc.DrawStringAnchored(d.weatherAPIService.Weather[0].Main, 170, 32, 0.5, 0.5)

	// Draw temp
	temp := fmt.Sprintf("%v°", math.Floor(d.weatherAPIService.Main.Temp))
	setFont(dc, 32)
	dc.DrawStringAnchored(temp, 170, 74, 0.5, 0.5)

	return dc.Image(), nil
}
