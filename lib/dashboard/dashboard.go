package dashboard

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"

	"github.com/briandowns/openweathermap"
	owm "github.com/briandowns/openweathermap"
	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	epd "github.com/justmiles/epd/lib/epd7in5v2"
	mdpng "github.com/justmiles/epd/lib/md-png"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	epdWidth  int = 800
	epdHeight int = 480
)

// Dashboard creates a custom Dashboard
type Dashboard struct {
	// Configure EPD
	Device     string
	EPDService *epd.EPD

	// WeatherAPI
	weatherAPIOptions *WeatherAPIOptions
	weatherAPIService *openweathermap.CurrentWeatherData

	// Calendar Location
	location string
}

// Options provides options for a new Dashboard
type Options func(d *Dashboard)

// NewDashboard creates a custom dashboard
func NewDashboard(opts ...Options) (*Dashboard, error) {
	var err error

	var d = &Dashboard{}
	for _, opt := range opts {
		opt(d)
	}

	// init EPD
	if d.Device != "" {

		if d.Device != "epd7in5v2" {
			return nil, fmt.Errorf("Device %s is not supported", d.Device)
		}

		d.EPDService, err = epd.NewRaspberryPiHat()
		if err != nil {
			return nil, err
		}
	}

	// init weatherAPI
	if d.weatherAPIOptions != nil {
		d.weatherAPIService, err = owm.NewCurrent(d.weatherAPIOptions.WeatherTempUnit, d.weatherAPIOptions.WeatherLanguage, d.weatherAPIOptions.WeatherAPIKey)
		if err != nil {
			return nil, fmt.Errorf("could not initialize weather api: %s", err)
		}
	}

	return d, nil

}

// ┌──────────────────────────────────────────────────────────────────────────────────────────────────────┐
// │┌────────────────────────────┐┌──────────────────────────────────────────────────────────────────────┐│
// ││                            ││                                                                      ││
// ││        <day of week>       ││  <HEADER>                                                            ││
// ││                            ││                                                                      ││
// ││       <Day  of Month>      │└──────────────────────────────────────────────────────────────────────┘│
// ││                            ││ <BODY>                                                               ││
// ││                            ││                                                                      ││
// ││    <Month>        <Year>   ││                                                                      ││
// ││ ────────────────────────── ││                                                                      ││
// ││                            ││                                                                      ││
// ││                 <WEATHER>  ││                                                                      ││
// ││ <WEATHER>                  ││                                                                      ││
// ││  <ICON>           <TEMP>   ││                                                                      ││
// ││                            ││                                                                      ││
// ││                            ││                                                                      ││
// ││                            ││                                                                      ││
// ││                            ││                                                                      ││
// ││                            ││                                                                      ││
// ││                            ││                                                                      ││
// ││                            ││                                                                      ││
// ││                            ││                                                                      ││
// ││                            ││                                                                      ││
// ││                            ││                                                                      ││
// ││                            ││                                                                      ││
// ││                            ││                                                                      ││
// │└────────────────────────────┘└──────────────────────────────────────────────────────────────────────┘│
// └──────────────────────────────────────────────────────────────────────────────────────────────────────┘
// Generate a dashboard
func (d *Dashboard) Generate(outputFile string, headerText string, bodyText string) error {
	var (
		xWidth, xHeight = float64(epdWidth), float64(epdHeight)
		oneSmeckle      = xWidth / 500
		err             error
	)
	// 800 x 480
	dc := gg.NewContext(epdWidth, epdHeight)

	// set white background
	dc.DrawRectangle(0, 0, xWidth, xHeight)
	dc.SetRGB(1, 1, 1)
	dc.Fill()

	// Draw the calendar widget
	calendarWidgetWidth := int(xWidth * .32)
	calendarWidgetHeight := int(xHeight * .45)
	cal, _ := buildCalendarWidget(calendarWidgetWidth, calendarWidgetHeight, d.location)
	dc.DrawImage(cal, 0, 0)

	// Draw the weather widget
	weatherWidgetWidth := int(xWidth * .32)
	weatherWidgetHeight := int(xHeight) - calendarWidgetHeight
	if d.weatherAPIOptions != nil {
		cal, err = d.buildWeatherWidget(weatherWidgetWidth, weatherWidgetHeight)
		if err != nil {
			return fmt.Errorf("could not build weather widget: %s", err)
		}
		dc.DrawImage(cal, 0, calendarWidgetHeight)
	}

	// Draw the dashboard header background
	taskHeaderStart := float64(calendarWidgetWidth) + oneSmeckle
	taskHeaderWidth := xWidth - taskHeaderStart
	taskHeaderHeight := xWidth * .08
	dc.DrawRectangle(taskHeaderStart, 0, taskHeaderWidth, taskHeaderHeight)
	dc.SetRGB(0, 0, 0)
	dc.Fill()

	// Draw the dashboard header text
	setFont(dc, taskHeaderHeight/2)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(headerText, taskHeaderStart+taskHeaderWidth/2, taskHeaderHeight/2, 0.5, 0.25)

	dc.SetRGB(0, 0, 0)

	bodyX := taskHeaderStart + oneSmeckle
	bodyY := taskHeaderHeight + oneSmeckle
	bodyWidth := taskHeaderWidth
	bodyHeight := xHeight - taskHeaderHeight

	// Convert body text to an image using mdpng
	var bodyBuf bytes.Buffer
	err = mdpng.Convert([]byte(bodyText), &bodyBuf,
		mdpng.WithWidth(int(bodyWidth)),
		mdpng.WithHeight(int(bodyHeight)),
		mdpng.WithFontSize(20),
	)
	if err != nil {
		return fmt.Errorf("could not render body text: %s", err)
	}

	bodyImg, err := png.Decode(&bodyBuf)
	if err != nil {
		return fmt.Errorf("could not decode body image: %s", err)
	}
	dc.DrawImage(bodyImg, int(bodyX), int(bodyY))

	// Save the image
	dc.SavePNG(outputFile)
	return nil
}

// DisplayImage accepts a path to image file and displays it on the screen
func (d *Dashboard) DisplayImage(filePath string) error {

	// If this is a URL, let's download it
	if isValidURL(filePath) {
		tmpfile, err := os.CreateTemp("", "epd	-image")
		if err != nil {
			return err
		}
		defer os.Remove(tmpfile.Name()) // clean up

		err = downloadFile(filePath, tmpfile.Name())
		if err != nil {
			return err
		}
		filePath = tmpfile.Name()
	}

	img, err := d.getImageFromFilePath(filePath)

	if err != nil {
		return err
	}

	buf := d.convertImage(img)
	d.EPDService.Display(buf)
	return nil
}

// DisplayText accepts a string text and displays it on the screen
func (d *Dashboard) DisplayText(text string) error {

	// Create new logo context
	dc := gg.NewContext(d.EPDService.Width, d.EPDService.Height)

	// Set Background Color
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// Set font color
	dc.SetColor(color.Black)

	dc.Fill()
	dc.SetRGB(0, 0, 0)

	maxWidth, maxHeight := float64(d.EPDService.Width), float64(d.EPDService.Height)

	fontSize, measuredHeight, err := fitTextToArea(dc, text, maxWidth, maxHeight)
	if err != nil {
		return fmt.Errorf("unable to fit text on screen: \n %s", text)
	}

	dc.DrawStringWrapped(text, 0, (maxHeight-measuredHeight)/2-(fontSize/4), 0, 0, maxWidth, 1, gg.AlignCenter)
	buf := d.convertImage(dc.Image())

	d.EPDService.Display(buf)

	return nil
}

// fitTextToArea dynamically adjusts the font size on the given context until the text
// fits within maxWidth x maxHeight. It returns the final fontSize, measuredHeight, and
// any error if the text cannot fit. The font face is set on dc upon return.
func fitTextToArea(dc *gg.Context, text string, maxWidth, maxHeight float64) (float64, float64, error) {
	var (
		fontSize          float64 = 300  // initial font size
		fontSizeReduction float64 = 0.95 // reduce the font size by this much until message fits in the display
		fontSizeMinimum   float64 = 10   // Smallest font size before giving up
		lineSpacing       float64 = 1
		measuredWidth     float64
		measuredHeight    float64
	)

	font, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return 0, 0, err
	}
	for {
		face := truetype.NewFace(font, &truetype.Options{Size: fontSize})
		dc.SetFontFace(face)

		wrappedLines := dc.WordWrap(text, maxWidth)
		wrappedText := strings.Join(wrappedLines, "\n")

		measuredWidth, measuredHeight = dc.MeasureMultilineString(wrappedText, lineSpacing)

		// If the message fits within the area, break. Otherwise reduce the font size and try again
		if measuredWidth <= maxWidth && measuredHeight <= maxHeight {
			break
		} else {
			fontSize = fontSize * fontSizeReduction
		}

		if fontSize < fontSizeMinimum {
			return 0, 0, fmt.Errorf("unable to fit text in area (%.0fx%.0f)", maxWidth, maxHeight)
		}
	}

	return fontSize, measuredHeight, nil
}

// fitPreformattedTextToArea dynamically adjusts the font size on the given context until
// all pre-formatted lines fit within maxWidth x maxHeight. Unlike fitTextToArea, it does
// not re-wrap text — each line is measured individually, preserving original line breaks
// and indentation.
func fitPreformattedTextToArea(dc *gg.Context, lines []string, maxWidth, maxHeight float64) (float64, float64, error) {
	var (
		fontSize          float64 = 300
		fontSizeReduction float64 = 0.95
		fontSizeMinimum   float64 = 10
		lineSpacing       float64 = 1.4
	)

	font, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return 0, 0, err
	}

	for {
		face := truetype.NewFace(font, &truetype.Options{Size: fontSize})
		dc.SetFontFace(face)

		totalHeight := float64(len(lines)) * fontSize * lineSpacing
		maxLineWidth := 0.0
		for _, line := range lines {
			w, _ := dc.MeasureString(line)
			if w > maxLineWidth {
				maxLineWidth = w
			}
		}

		if maxLineWidth <= maxWidth && totalHeight <= maxHeight {
			return fontSize, totalHeight, nil
		}

		fontSize = fontSize * fontSizeReduction
		if fontSize < fontSizeMinimum {
			return 0, 0, fmt.Errorf("unable to fit text in area (%.0fx%.0f)", maxWidth, maxHeight)
		}
	}
}

func (d *Dashboard) getImageFromFilePath(filePath string) (image.Image, error) {

	img, err := imaging.Open(filePath, imaging.AutoOrientation(true))
	if err != nil {
		return nil, err
	}

	// Rotate if necessary
	if img.Bounds().Max.X == d.EPDService.Height && img.Bounds().Max.Y == d.EPDService.Width {
		img = imaging.Rotate90(img)
	}

	// Resize the image to match current dimensions
	img = imaging.Resize(img, d.EPDService.Width, d.EPDService.Height, imaging.Lanczos)

	// GreyScale the image
	img = imaging.Grayscale(img)
	img = imaging.AdjustContrast(img, 20)
	img = imaging.Sharpen(img, 2)

	return img, err
}

// Convert converts the input image into a ready-to-display byte buffer.
func (d *Dashboard) convertImage(img image.Image) []byte {
	var byteToSend byte = 0x00
	var bgColor = 1

	buffer := bytes.Repeat([]byte{byteToSend}, (d.EPDService.Width/8)*d.EPDService.Height)

	for j := 0; j < d.EPDService.Height; j++ {
		for i := 0; i < d.EPDService.Width; i++ {
			bit := bgColor

			if i < img.Bounds().Dx() && j < img.Bounds().Dy() {
				bit = color.Palette([]color.Color{color.White, color.Black}).Index(img.At(i, j))
			}

			if bit == 1 {
				byteToSend |= 0x80 >> (uint32(i) % 8)
			}

			if i%8 == 7 {
				buffer[(i/8)+(j*(d.EPDService.Width/8))] = byteToSend
				byteToSend = 0x00
			}
		}
	}

	return buffer
}
