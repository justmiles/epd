package epd7in5v2

// ported from https://github.com/waveshare/e-Paper/blob/master/RaspberryPi%26JetsonNano/c/lib/e-Paper/EPD_7in5_V2.c

import (
	"image/color"
	"strings"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	rpio "github.com/stianeikeland/go-rpio/v4"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	epdWidth  int = 800
	epdHeight int = 480
)

// EPD ..
type EPD struct {
	resetPin uint8
	dcPin    uint8
	csPin    uint8
	busyPin  uint8
}

// NewRaspberryPiHat intialized the EDP for Raspberry PI
func NewRaspberryPiHat() (*EPD, error) {
	return New(
		17, // Reset Pin / GPIO_17
		25, // DC Pin / GPIO_25
		8,  // CS Pin / GPIO_8
		24, // Busy Pin / GPIO_24
	)
}

// New EPD7in5_V2 str
func New(resetPin, dcPin, csPin, busyPin uint8) (*EPD, error) {
	err := rpio.Open()
	if err != nil {
		return nil, err
	}

	return &EPD{
		resetPin: resetPin,
		dcPin:    dcPin,
		csPin:    csPin,
		busyPin:  busyPin,
	}, nil
}

// Display sends the image buffer in RAM to e-Paper and display
func (epd EPD) Display(img []byte) {
	epd.SendCommand(0x13)
	delayMS(2)

	for i := 0; i < len(img); i++ {
		epd.SendData(img[i])
	}

	epd.TurnOnDisplay()
}

// TurnOnDisplay turns on the device display
func (epd EPD) TurnOnDisplay() {
	epd.SendCommand(0x12)
	delayMS(100)
	epd.ReadBusy()
}

// Clear the display
func (epd EPD) Clear() {
	epd.SendCommand(0x10)

	for i := 1; i <= int(epdWidth*epdHeight/8); i++ {
		epd.SendData(0x00)
	}

	epd.SendCommand(0x13)
	for i := 1; i <= int(epdWidth*epdHeight/8); i++ {
		epd.SendData(0x00)
	}

	epd.TurnOnDisplay()
}

// HardwareReset resets the hardware
func (epd EPD) HardwareReset() {
	digitalWrite(epd.resetPin, rpio.High)
	delayMS(200)
	digitalWrite(epd.resetPin, rpio.Low)
	delayMS(2)
	digitalWrite(epd.resetPin, rpio.High)
	delayMS(200)
}

// SendCommand to device
func (epd EPD) SendCommand(command ...byte) {
	digitalWrite(epd.dcPin, rpio.Low)
	digitalWrite(epd.csPin, rpio.Low)
	spiWrite(command...)
	digitalWrite(epd.csPin, rpio.High)
}

// SendData to device
func (epd EPD) SendData(command ...byte) {
	digitalWrite(epd.dcPin, rpio.High)
	digitalWrite(epd.csPin, rpio.Low)
	spiWrite(command...)
	digitalWrite(epd.csPin, rpio.High)
}

// HardwareInit inits the hardware
func (epd EPD) HardwareInit() {
	epd.HardwareReset()

	epd.SendCommand(0x01) // Power Setting
	epd.SendData(0x07)
	epd.SendData(0x07) // VGH=20V,VGL=-20V
	epd.SendData(0x3f) // VDH=15V
	epd.SendData(0x3f) // VDL=-15V

	epd.SendCommand(0x04) // POWER ON
	delayMS(100)
	epd.ReadBusy()

	epd.SendCommand(0x00) // PANNEL SETTING
	epd.SendData(0x1F)    // KW-3f   KWR-2F	BWROTP 0f	BWOTP 1f

	epd.SendCommand(0x61) // tres
	epd.SendData(0x03)    // source 800
	epd.SendData(0x20)
	epd.SendData(0x01) // gate 480
	epd.SendData(0xE0)

	epd.SendCommand(0x15)
	epd.SendData(0x00)

	epd.SendCommand(0x50) // VCOM AND DATA INTERVAL SETTING
	epd.SendData(0x10)
	epd.SendData(0x07)

	epd.SendCommand(0x60) // TCON SETTING
	epd.SendData(0x22)
}

// ReadBusy reads
func (epd EPD) ReadBusy() {
	epd.SendCommand(0x71)
	for digitalRead(epd.busyPin) == rpio.Low {
		// fmt.Println("e-Paper busy")
		epd.SendCommand(0x71)
		delayMS(200)
	}
}

// Sleep enters sleep mode
func (epd EPD) Sleep() {
	epd.SendCommand(0x02) // POWER_OFF
	epd.ReadBusy()

	epd.SendCommand(0x07) // DEEP_SLEEP
	epd.SendData(0xA5)
}

// DisplayImage an image file and displays on the screen
func (epd EPD) DisplayImage(filePath string) error {
	img, err := getImageFromFilePath(filePath)

	if err != nil {
		return err
	}

	buf := convertImage(img)
	epd.Display(buf)
	return nil
}

// DisplayText renders the contents on the screen
func (epd EPD) DisplayText(text string) error {

	// Create new logo context
	dc := gg.NewContext(epdWidth, epdHeight)

	// Set Background Color
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// Set font color
	dc.SetColor(color.Black)

	const padding float64 = 5 // padding

	dc.Fill()
	dc.SetRGB(0, 0, 0)

	var (
		yPad                = padding
		xWidth              = float64(epdWidth) - (padding * 2)
		fontSize    float64 = 100
		r           float64 = float64(epdHeight) / 10 // font reduction size
		lineSpacing float64 = 1.3
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
		verticalSpace := float64(epdHeight) - sh
		if verticalSpace/2 > padding {
			yPad = verticalSpace / 2
		}
		if sw < float64(epdWidth)-(2*padding) && sh <= float64(epdHeight)-padding {
			break
		}
		fontSize = fontSize - r
	}

	dc.DrawStringWrapped(text, padding, yPad, 0.0, 0.0, xWidth, lineSpacing, gg.AlignCenter)
	buf := convertImage(dc.Image())

	epd.Display(buf)

	return nil
}
