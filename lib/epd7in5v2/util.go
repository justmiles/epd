package epd7in5v2

import (
	"log"
	"os"
	"time"

	rpio "github.com/stianeikeland/go-rpio/v4"
)

func digitalWrite(i uint8, state rpio.State) {
	debug("util -> digitalWrite(%v,%v)", i, state)

	pin := rpio.Pin(i)
	pin.Write(state)
}

func digitalRead(i uint8) rpio.State {
	debug("util -> digitalRead(%v)", i)
	return rpio.ReadPin(rpio.Pin(i))
}

func delayMS(ms int64) {
	debug("util -> delayMS(%v)", ms)
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func spiWrite(command ...byte) {
	debug("util -> spiWrite(%v)", command)
	err := rpio.SpiBegin(rpio.Spi0)
	if err != nil {
		panic(err)
	}

	rpio.SpiTransmit(command...)

	rpio.SpiEnd(rpio.Spi0)

}

var l = log.New(os.Stdout, "", 0)

func debug(msg string, i ...interface{}) {
	// l.SetPrefix(time.Now().Format("2006-01-02 15:04:05 - "))
	// l.Print(fmt.Sprintf(msg, i...))
}
