package epd

// EPD - Electronic Paper Display struct
type EPD interface {
	Reset()
	SendCommand()
	SendData()
	ReadBusy()
	Init()
	Display()
	Clear()
	Sleep()
	Exit() // TODO
}
