package dashboard

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"os"
	"strings"

	"github.com/briandowns/openweathermap"
	owm "github.com/briandowns/openweathermap"
	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/jubnzv/go-taskwarrior"
	epd "github.com/justmiles/epd/lib/epd7in5v2"
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

	// TaskWarrior
	taskWarriorOptions *TaskWarriorOptions
	taskWarriorService *taskwarrior.TaskWarrior
}

// Options provides options for a new Dashboard
type Options func(cd *Dashboard)

// NewDashboard creates a custom dashboard
func NewDashboard(opts ...Options) (*Dashboard, error) {
	var err error

	var cd = &Dashboard{}
	for _, opt := range opts {
		opt(cd)
	}

	// init EPD
	cd.EPDService, err = epd.NewRaspberryPiHat()
	if err != nil {
		return nil, err
	}

	if cd.Device != "epd7in5v2" {
		return nil, fmt.Errorf("Device %s is not supported", cd.Device)
	}

	// init TaskWarrior
	if cd.taskWarriorOptions != nil {
		cd.taskWarriorService, err = taskwarrior.NewTaskWarrior(cd.taskWarriorOptions.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("could not initialize task warrior: %s", err)
		}
	}

	// init weatherAPI
	if cd.weatherAPIOptions != nil {
		cd.weatherAPIService, err = owm.NewCurrent(cd.weatherAPIOptions.WeatherTempUnit, cd.weatherAPIOptions.WeatherLanguage, cd.weatherAPIOptions.WeatherAPIKey)
		if err != nil {
			return nil, fmt.Errorf("could not initialize task warrior: %s", err)
		}
	}

	return cd, nil

}

// Generate a custom dashboard
func (cd *Dashboard) Generate(outputFile string) error {
	var (
		xWidth, xHeight = float64(epdWidth), float64(epdHeight)
		err             error
	)

	dc := gg.NewContext(epdWidth, epdHeight)

	// set white background
	dc.DrawRectangle(0, 0, xWidth, xHeight)
	dc.SetRGB(1, 1, 1)
	dc.Fill()

	// Draw the calendar widget
	cal, _ := buildCalendarWidget(256, 220)
	dc.DrawImage(cal, 0, 0)

	// Draw the weather widget
	if cd.weatherAPIOptions != nil {
		cal, err = cd.buildWeatherWidget(256, 260)
		if err != nil {
			return fmt.Errorf("could not build weather widget: %s", err)
		}
		dc.DrawImage(cal, 0, 220)
	}

	// Draw the Task header
	dc.DrawRectangle(266, 10, 524, 64)
	dc.SetRGB(0, 0, 0)
	dc.Fill()

	// Draw the Task header
	setFont(dc, 32)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored("TODOs", 276, 35, 0, .5)

	// Draw TaskWarrior
	if cd.taskWarriorService != nil {
		setFont(dc, 22)
		dc.SetRGB(0, 0, 0)
		var yPosition float64 = 64
		for _, task := range cd.getTaskWarriorTasks() {
			yPosition = yPosition + 32
			dc.DrawStringAnchored(task, 276, yPosition, 0, .5)

			dc.DrawRectangle(266, yPosition+16, 524, 2)
			dc.Fill()

			dc.DrawRectangle(266, yPosition, 2, 16)
			dc.Fill()
		}
	}

	// Save the image
	dc.SavePNG(outputFile)
	return nil
}

// DisplayImage accepts a path to image file and displays it on the screen
func (cd *Dashboard) DisplayImage(filePath string) error {

	// If this is a URL, let's download it
	if isValidURL(filePath) {
		tmpfile, err := ioutil.TempFile("", "epd-image")
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

	img, err := cd.getImageFromFilePath(filePath)

	if err != nil {
		return err
	}

	buf := cd.convertImage(img)
	cd.EPDService.Display(buf)
	return nil
}

// DisplayText accepts a string text and displays it on the screen
func (cd *Dashboard) DisplayText(text string) error {

	// Create new logo context
	dc := gg.NewContext(cd.EPDService.Width, cd.EPDService.Height)

	// Set Background Color
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// Set font color
	dc.SetColor(color.Black)

	dc.Fill()
	dc.SetRGB(0, 0, 0)

	var (
		maxWidth, maxHeight           float64 = float64(cd.EPDService.Width), float64(cd.EPDService.Height)
		fontSize                      float64 = 300  // initial font size
		fontSizeReduction             float64 = 0.95 // reduce the font size by this much until message fits in the display
		fontSizeMinimum               float64 = 10   // Smallest font size before giving up
		lineSpacing                   float64 = 1
		measuredWidth, measuredHeight float64
	)

	font, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return err
	}
	for {
		face := truetype.NewFace(font, &truetype.Options{Size: fontSize})
		dc.SetFontFace(face)

		stringLines := dc.WordWrap(text, maxWidth)

		measuredWidth, measuredHeight = dc.MeasureMultilineString(strings.Join(stringLines, "\n"), lineSpacing)

		// If the message fits within the frame, let's break. Otherwise reduce the font size and try again
		if measuredWidth < maxWidth && measuredHeight <= maxHeight {
			break
		} else {
			fontSize = fontSize * fontSizeReduction
		}

		if fontSize < fontSizeMinimum {
			return fmt.Errorf("unable to fit text on screen: \n %s", text)
		}
		// TODO: debug logging: fmt.Printf("font size: %v\n", fontSize)
	}

	dc.DrawStringWrapped(text, 0, (maxHeight-measuredHeight)/2-(fontSize/4), 0, 0, maxWidth, lineSpacing, gg.AlignCenter)
	buf := cd.convertImage(dc.Image())

	cd.EPDService.Display(buf)

	return nil
}

func (cd *Dashboard) getImageFromFilePath(filePath string) (image.Image, error) {

	img, err := imaging.Open(filePath, imaging.AutoOrientation(true))
	if err != nil {
		return nil, err
	}

	// Rotate if necessary
	if img.Bounds().Max.X == cd.EPDService.Height && img.Bounds().Max.Y == cd.EPDService.Width {
		img = imaging.Rotate90(img)
	}

	// Resize the image to match current dimensions
	img = imaging.Resize(img, cd.EPDService.Width, cd.EPDService.Height, imaging.Lanczos)

	// GreyScale the image
	img = imaging.Grayscale(img)
	img = imaging.AdjustContrast(img, 20)
	img = imaging.Sharpen(img, 2)

	return img, err
}

// Convert converts the input image into a ready-to-display byte buffer.
func (cd *Dashboard) convertImage(img image.Image) []byte {
	var byteToSend byte = 0x00
	var bgColor = 1

	buffer := bytes.Repeat([]byte{byteToSend}, (cd.EPDService.Width/8)*cd.EPDService.Height)

	for j := 0; j < cd.EPDService.Height; j++ {
		for i := 0; i < cd.EPDService.Width; i++ {
			bit := bgColor

			if i < img.Bounds().Dx() && j < img.Bounds().Dy() {
				bit = color.Palette([]color.Color{color.White, color.Black}).Index(img.At(i, j))
			}

			if bit == 1 {
				byteToSend |= 0x80 >> (uint32(i) % 8)
			}

			if i%8 == 7 {
				buffer[(i/8)+(j*(cd.EPDService.Width/8))] = byteToSend
				byteToSend = 0x00
			}
		}
	}

	return buffer
}
