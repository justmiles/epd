package epd7in5v2

import (
	"bytes"
	"time"

	"image"
	"image/color"

	"github.com/disintegration/imaging"
	rpio "github.com/stianeikeland/go-rpio/v4"
)

func digitalWrite(i uint8, state rpio.State) {
	pin := rpio.Pin(i)
	pin.Write(state)
}

func digitalRead(i uint8) rpio.State {
	return rpio.ReadPin(rpio.Pin(i))
}

func delayMS(ms int64) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func spiWrite(command ...byte) {

	rpio.SpiSpeed(4000000)
	rpio.SpiMode(0, 0)
	err := rpio.SpiBegin(rpio.Spi0)
	if err != nil {
		panic(err)
	}

	rpio.SpiTransmit(command...)

	err = rpio.SpiBegin(rpio.Spi0)
	if err != nil {
		panic(err)
	}
}

func getImageFromFilePath(filePath string) (image.Image, error) {

	img, err := imaging.Open(filePath, imaging.AutoOrientation(true))
	if err != nil {
		return nil, err
	}

	// Rotate if necessary
	if img.Bounds().Max.X == epdHeight && img.Bounds().Max.Y == epdWidth {
		img = imaging.Rotate90(img)
	}

	// Resize the image to match current dimensions
	img = imaging.Resize(img, epdWidth, epdHeight, imaging.Lanczos)

	// GreyScale the image
	img = imaging.Grayscale(img)
	img = imaging.AdjustContrast(img, 20)
	img = imaging.Sharpen(img, 2)

	return img, err
}

// Convert converts the input image into a ready-to-display byte buffer.
func convertImage(img image.Image) []byte {
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
