package dashboard

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
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

	const padding float64 = 5 // padding

	var (
		yPad                      = padding
		xWidth                    = float64(d.E.Width) - (padding * 2)
		fontSize          float64 = 300 // initial font size
		fontSizeReduction float64 = 10  // reduce the font size by this much until message fits in the display
		lineSpacing       float64 = 1.5
	)

	font, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return err
	}

	for {
		face := truetype.NewFace(font, &truetype.Options{Size: fontSize})
		dc.SetFontFace(face)

		stringLines := dc.WordWrap(text, xWidth)

		sw, sh := dc.MeasureMultilineString(strings.Join(stringLines, "\n"), lineSpacing)
		verticalSpace := float64(d.E.Height) - sh
		if verticalSpace/2 > padding {
			yPad = verticalSpace / 2
		}
		if sw < float64(d.E.Width)-(2*padding) && sh <= float64(d.E.Height)-padding {
			break
		}
		fontSize = fontSize - fontSizeReduction

		if fontSize < fontSizeReduction {
			return fmt.Errorf("unable to fit text on screen: \n %s", text)
		}
		// TODO: debug logging: fmt.Printf("font size: %v\n", fontSize)
	}

	dc.DrawStringWrapped(text, padding, yPad, 0.0, 0.0, xWidth, lineSpacing, gg.AlignCenter)
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
