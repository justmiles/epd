package epd7in5v2

// ported from https://github.com/waveshare/e-Paper/blob/master/RaspberryPi%26JetsonNano/c/lib/e-Paper/EPD_7in5_V2.c

import (
	"fmt"
	"os"
	"time"

	rpio "github.com/stianeikeland/go-rpio/v4"
)

const (
	debugEnabled bool = false

	epdWidth  int = 800
	epdHeight int = 480

	// Byte settings pulled from
	// https://github.com/waveshare/e-Paper/blob/751a9fb93fdd486511222777b0070c51bf436386/RaspberryPi%26JetsonNano/c/lib/e-Paper/EPD_7in5.c#L25
	panelSetting                 byte = 0x00
	powerSetting                 byte = 0x01
	powerOff                     byte = 0x02
	powerOffSequenceSetting      byte = 0x03
	powerOn                      byte = 0x04
	powerOnMeasure               byte = 0x05
	boosterSoftStart             byte = 0x06
	deepSleep                    byte = 0x07
	dataStartTransmission1       byte = 0x10
	dataStop                     byte = 0x11
	displayRefresh               byte = 0x12
	dataStartTransmission2       byte = 0x13
	vcomLut                      byte = 0x20
	w2WLut                       byte = 0x21
	b2WLut                       byte = 0x22
	w2BLut                       byte = 0x23
	b2BLut                       byte = 0x24
	pllControl                   byte = 0x30
	temperatureSensorCalibration byte = 0x40
	temperatureSensorSelection   byte = 0x41
	temperatureSensorWrite       byte = 0x42
	temperatureSensorRead        byte = 0x43
	vcomAndDataIntervalSetting   byte = 0x50
	lowPowerDetection            byte = 0x51
	tconSetting                  byte = 0x60
	resolutionSetting            byte = 0x61
	getStatus                    byte = 0x71
	autoMeasureVcom              byte = 0x80
	readVcomValue                byte = 0x81
	vcmDcSetting                 byte = 0x82
	partialWindow                byte = 0x90
	partialIn                    byte = 0x91
	partialOut                   byte = 0x92
	programMode                  byte = 0xa0
	activeProgram                byte = 0xa1
	readOtpData                  byte = 0xa2
	powerSaving                  byte = 0xe3
)

// EPD ..
type EPD struct {
	resetPin uint8
	dcPin    uint8
	csPin    uint8
	busyPin  uint8

	Height int
	Width  int
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

	// rpio.Mode()
	err := rpio.Open()
	if err != nil {
		return nil, err
	}

	rpio.PinMode(rpio.Pin(resetPin), rpio.Mode(rpio.Output))
	rpio.PinMode(rpio.Pin(dcPin), rpio.Mode(rpio.Output))
	rpio.PinMode(rpio.Pin(csPin), rpio.Mode(rpio.Output))
	rpio.PinMode(rpio.Pin(busyPin), rpio.Mode(rpio.Input))

	rpio.SpiSpeed(4000000)
	rpio.SpiChipSelect(0)

	return &EPD{
		resetPin: resetPin,
		dcPin:    dcPin,
		csPin:    csPin,
		busyPin:  busyPin,
		Height:   epdHeight,
		Width:    epdWidth,
	}, nil
}

// Display is used to transmit a frame of image and display
func (epd EPD) Display(img []byte) {
	epd.SendCommand(dataStartTransmission2)
	delayMS(2)

	for i := 0; i < len(img); i++ {
		epd.SendData(img[i])
	}

	epd.TurnOnDisplay()
}

// TurnOnDisplay turns on the device display
func (epd EPD) TurnOnDisplay() {
	epd.SendCommand(displayRefresh)
	delayMS(100)
	epd.ReadBusy()
}

// Clear is used to clear the e-paper to white
func (epd EPD) Clear() {
	epd.SendCommand(dataStartTransmission1)

	for i := 1; i <= int(epdWidth*epdHeight/8); i++ {
		epd.SendData(panelSetting)
	}

	epd.SendCommand(dataStartTransmission2)
	for i := 1; i <= int(epdWidth*epdHeight/8); i++ {
		epd.SendData(panelSetting)
	}

	epd.TurnOnDisplay()
}

// HardwareReset resets the hardware
func (epd EPD) HardwareReset() {
	debug("epd -> HardwareReset")
	digitalWrite(epd.resetPin, rpio.High)
	delayMS(200)
	digitalWrite(epd.resetPin, rpio.Low)
	delayMS(2)
	digitalWrite(epd.resetPin, rpio.High)
	delayMS(200)
}

// SendCommand to device
func (epd EPD) SendCommand(command ...byte) {
	debug("epd -> SendCommand %v", command)

	digitalWrite(epd.dcPin, rpio.Low)
	digitalWrite(epd.csPin, rpio.Low)
	spiWrite(command...)
	digitalWrite(epd.csPin, rpio.High)
}

// SendData to device
func (epd EPD) SendData(command ...byte) {
	debug("epd -> SendData %v", command)

	digitalWrite(epd.dcPin, rpio.High)
	digitalWrite(epd.csPin, rpio.Low)
	spiWrite(command...)
	digitalWrite(epd.csPin, rpio.High)
}

// HardwareInit used to initialize e-Paper or wakeup e-Paper from sleep mode.
func (epd EPD) HardwareInit() {
	debug("epd -> HardwareInit")

	epd.HardwareReset()

	epd.SendCommand(powerSetting)
	epd.SendData(deepSleep)
	epd.SendData(deepSleep)
	epd.SendData(0x3f) // VDH=15V
	epd.SendData(0x3f) // VDL=-15V
	delayMS(200)

	epd.SendCommand(powerOn)
	delayMS(100)
	epd.ReadBusy()
	delayMS(200)

	epd.SendCommand(panelSetting)
	epd.SendData(0x1F) // KW-3f   KWR-2F	BWROTP 0f	BWOTP 1f
	delayMS(200)

	epd.SendCommand(resolutionSetting) // tres
	epd.SendData(powerOffSequenceSetting)
	epd.SendData(vcomLut)
	epd.SendData(powerSetting) // gate 480
	epd.SendData(0xE0)
	delayMS(200)

	epd.SendCommand(0x15)
	epd.SendData(panelSetting)
	delayMS(200)

	epd.SendCommand(vcomAndDataIntervalSetting) // VCOM AND DATA INTERVAL SETTING
	epd.SendData(dataStartTransmission1)
	epd.SendData(deepSleep)
	delayMS(200)

	epd.SendCommand(tconSetting)
	epd.SendData(b2WLut)
}

// ReadBusy reads
func (epd EPD) ReadBusy() {

	// Setup Timeout
	ch := make(chan bool, 1)
	timeout := make(chan bool, 1)
	defer close(ch)
	defer close(timeout)

	// Timeout function
	go func() {
		time.Sleep(60 * time.Second)
		timeout <- true
	}()

	// Wait for busy
	go func() {

		epd.SendCommand(getStatus)
		for digitalRead(epd.busyPin) == rpio.Low {
			debug("epd -> ReadBusy")

			epd.SendCommand(getStatus)
			delayMS(200)
		}

		ch <- true
	}()

	select {
	case <-ch:
		return
	case <-timeout:
		fmt.Println("Timeout waiting for EPD busy status. Did you inititalize (wake) the device?")
		os.Exit(2)
	}
}

// Sleep is used to set the device to sleep mode
func (epd EPD) Sleep() {
	epd.SendCommand(powerOff)
	epd.ReadBusy()

	epd.SendCommand(deepSleep)
	epd.SendData(0xA5)
}
