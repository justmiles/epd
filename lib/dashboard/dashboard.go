package dashboard

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	epd "github.com/justmiles/epd/lib/epd7in5v2"
	"golang.org/x/image/font/gofont/goregular"
)

type Dashboard struct {
	E *epd.EPD
}

// NewDashboard returns a new dashboard
func NewDashboard(device string) (*Dashboard, error) {
	var (
		d   Dashboard
		err error
	)

	d.E, err = epd.NewRaspberryPiHat()
	if err != nil {
		return nil, err
	}

	if device != "epd7in5v2" {
		return nil, fmt.Errorf("Device %s is not supported", device)
	}

	return &d, nil

}

// DisplayImage accepts a path to image file and displays it on the screen
func (d *Dashboard) DisplayImage(filePath string) error {

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

	img, err := d.getImageFromFilePath(filePath)

	if err != nil {
		return err
	}

	buf := d.convertImage(img)
	d.E.Display(buf)
	return nil
}

// DisplayText accepts a string text and displays it on the screen
func (d *Dashboard) DisplayText(text string) error {

	// Create new logo context
	dc := gg.NewContext(d.E.Width, d.E.Height)

	// Set Background Color
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// Set font color
	dc.SetColor(color.Black)

	dc.Fill()
	dc.SetRGB(0, 0, 0)

	var (
		maxWidth, maxHeight           float64 = float64(d.E.Width), float64(d.E.Height)
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
	buf := d.convertImage(dc.Image())

	d.E.Display(buf)

	return nil
}

func (d *Dashboard) getImageFromFilePath(filePath string) (image.Image, error) {

	img, err := imaging.Open(filePath, imaging.AutoOrientation(true))
	if err != nil {
		return nil, err
	}

	// Rotate if necessary
	if img.Bounds().Max.X == d.E.Height && img.Bounds().Max.Y == d.E.Width {
		img = imaging.Rotate90(img)
	}

	// Resize the image to match current dimensions
	img = imaging.Resize(img, d.E.Width, d.E.Height, imaging.Lanczos)

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

	buffer := bytes.Repeat([]byte{byteToSend}, (d.E.Width/8)*d.E.Height)

	for j := 0; j < d.E.Height; j++ {
		for i := 0; i < d.E.Width; i++ {
			bit := bgColor

			if i < img.Bounds().Dx() && j < img.Bounds().Dy() {
				bit = color.Palette([]color.Color{color.White, color.Black}).Index(img.At(i, j))
			}

			if bit == 1 {
				byteToSend |= 0x80 >> (uint32(i) % 8)
			}

			if i%8 == 7 {
				buffer[(i/8)+(j*(d.E.Width/8))] = byteToSend
				byteToSend = 0x00
			}
		}
	}

	return buffer
}

func downloadFile(fullURLFIle, localFilePath string) (err error) {

	// Create blank file
	file, err := os.Create(localFilePath)
	if err != nil {
		return fmt.Errorf("Error creating temporary file:\n %s", err)
	}

	// Put content on file
	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	// Invoke HTTP request
	resp, err := client.Get(fullURLFIle)
	defer resp.Body.Close()
	if err != nil {
		return fmt.Errorf("Error downloading logo:\n %s", err)
	}

	// Pipe data to disk
	_, err = io.Copy(file, resp.Body)
	defer file.Close()
	if err != nil {
		return fmt.Errorf("Error writing logo to disk:\n %s", err)
	}

	return nil
}

// isValidURL determins if a string is an actual URL
func isValidURL(toTest string) bool {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}

	u, err := url.Parse(toTest)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}
