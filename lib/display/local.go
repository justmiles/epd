package display

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
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

// LocalDisplay implements Service for direct hardware access via SPI/GPIO.
type LocalDisplay struct {
	epd    *epd.EPD
	device string
}

// NewLocalDisplay creates a new local display service for the given device.
func NewLocalDisplay(device string) (*LocalDisplay, error) {
	if device != "epd7in5v2" {
		return nil, fmt.Errorf("device %s is not supported", device)
	}

	epdDevice, err := epd.NewRaspberryPiHat()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize EPD hardware: %w", err)
	}

	return &LocalDisplay{
		epd:    epdDevice,
		device: device,
	}, nil
}

// HardwareInit initializes (wakes) the display hardware.
func (l *LocalDisplay) HardwareInit() {
	l.epd.HardwareInit()
}

// DisplayImage accepts raw PNG data and displays it on the EPD.
func (l *LocalDisplay) DisplayImage(pngData []byte) error {
	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return fmt.Errorf("failed to decode PNG: %w", err)
	}

	processed := processImage(img, l.epd.Width, l.epd.Height)
	buf := convertImage(processed, l.epd.Width, l.epd.Height)
	l.epd.Display(buf)
	return nil
}

// DisplayImageFromFile reads an image from a file path or URL and displays it.
func (l *LocalDisplay) DisplayImageFromFile(filePath string) error {
	pngData, err := ReadImageFile(filePath, l.epd.Width, l.epd.Height)
	if err != nil {
		return err
	}
	return l.DisplayImage(pngData)
}

// DisplayText renders text and displays it on the EPD.
func (l *LocalDisplay) DisplayText(text string) error {
	img, err := renderText(text, l.epd.Width, l.epd.Height)
	if err != nil {
		return err
	}

	buf := convertImage(img, l.epd.Width, l.epd.Height)
	l.epd.Display(buf)
	return nil
}

// Clear clears the EPD to white.
func (l *LocalDisplay) Clear() error {
	l.epd.Clear()
	return nil
}

// Sleep puts the EPD into sleep mode.
func (l *LocalDisplay) Sleep() error {
	l.epd.Sleep()
	return nil
}

// Close releases resources. For local display, this is a no-op.
func (l *LocalDisplay) Close() error {
	return nil
}

// EPD returns the underlying EPD device for direct access (e.g. HardwareInit).
func (l *LocalDisplay) EPD() *epd.EPD {
	return l.epd
}

// --- Shared image processing utilities ---

// ReadImageFile reads an image from a file path or URL and returns PNG-encoded bytes.
func ReadImageFile(filePath string, width, height int) ([]byte, error) {
	// If this is a URL, download it
	if isValidURL(filePath) {
		tmpfile, err := ioutil.TempFile("", "epd-image")
		if err != nil {
			return nil, err
		}
		defer os.Remove(tmpfile.Name())

		err = downloadFile(filePath, tmpfile.Name())
		if err != nil {
			return nil, err
		}
		filePath = tmpfile.Name()
	}

	img, err := imaging.Open(filePath, imaging.AutoOrientation(true))
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}

	processed := processImage(img, width, height)

	var buf bytes.Buffer
	if err := png.Encode(&buf, processed); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buf.Bytes(), nil
}

func processImageNRGBA(img *image.NRGBA, width, height int) *image.NRGBA {
	// Rotate if necessary
	if img.Bounds().Max.X == height && img.Bounds().Max.Y == width {
		img = imaging.Rotate90(img)
	}

	// Resize to match display dimensions
	img = imaging.Resize(img, width, height, imaging.Lanczos)

	// Greyscale and enhance
	img = imaging.Grayscale(img)
	img = imaging.AdjustContrast(img, 20)
	img = imaging.Sharpen(img, 2)

	return img
}

func processImage(img image.Image, width, height int) image.Image {
	nrgba := imaging.Clone(img)
	return processImageNRGBA(nrgba, width, height)
}

func renderText(text string, epdWidth, epdHeight int) (image.Image, error) {
	dc := gg.NewContext(epdWidth, epdHeight)

	// Set Background Color
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// Set font color
	dc.SetColor(color.Black)
	dc.Fill()
	dc.SetRGB(0, 0, 0)

	var (
		maxWidth, maxHeight           float64 = float64(epdWidth), float64(epdHeight)
		fontSize                      float64 = 300
		fontSizeReduction             float64 = 0.95
		fontSizeMinimum               float64 = 10
		lineSpacing                   float64 = 1
		measuredWidth, measuredHeight float64
	)

	font, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}
	for {
		face := truetype.NewFace(font, &truetype.Options{Size: fontSize})
		dc.SetFontFace(face)

		stringLines := dc.WordWrap(text, maxWidth)
		measuredWidth, measuredHeight = dc.MeasureMultilineString(strings.Join(stringLines, "\n"), lineSpacing)

		if measuredWidth < maxWidth && measuredHeight <= maxHeight {
			break
		} else {
			fontSize = fontSize * fontSizeReduction
		}

		if fontSize < fontSizeMinimum {
			return nil, fmt.Errorf("unable to fit text on screen: \n %s", text)
		}
	}

	dc.DrawStringWrapped(text, 0, (maxHeight-measuredHeight)/2-(fontSize/4), 0, 0, maxWidth, lineSpacing, gg.AlignCenter)
	return dc.Image(), nil
}

// convertImage converts an image into a ready-to-display byte buffer for the EPD.
func convertImage(img image.Image, epdWidth, epdHeight int) []byte {
	var byteToSend byte = 0x00
	var bgColor = 1

	buffer := bytes.Repeat([]byte{byteToSend}, (epdWidth/8)*epdHeight)

	for j := 0; j < epdHeight; j++ {
		for i := 0; i < epdWidth; i++ {
			bit := bgColor

			if i < img.Bounds().Dx() && j < img.Bounds().Dy() {
				bit = color.Palette([]color.Color{color.White, color.Black}).Index(img.At(i, j))
			}

			if bit == 1 {
				byteToSend |= 0x80 >> (uint32(i) % 8)
			}

			if i%8 == 7 {
				buffer[(i/8)+(j*(epdWidth/8))] = byteToSend
				byteToSend = 0x00
			}
		}
	}

	return buffer
}

func downloadFile(fullURLFile, localFilePath string) error {
	file, err := os.Create(localFilePath)
	if err != nil {
		return fmt.Errorf("error creating temporary file: %w", err)
	}
	defer file.Close()

	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	resp, err := client.Get(fullURLFile)
	if err != nil {
		return fmt.Errorf("error downloading image: %w", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("error writing image to disk: %w", err)
	}

	return nil
}

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
