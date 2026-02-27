package display

import "strings"

// Service abstracts over local hardware and remote gRPC display operations.
type Service interface {
	// DisplayImage accepts raw PNG data and displays it on the EPD.
	DisplayImage(pngData []byte) error

	// DisplayText renders text and displays it on the EPD.
	DisplayText(text string) error

	// Clear clears the EPD to white.
	Clear() error

	// Sleep puts the EPD into sleep mode.
	Sleep() error

	// Close releases any resources held by the display service.
	Close() error
}

// IsRemote returns true if the device string looks like a remote host:port address.
func IsRemote(device string) bool {
	return strings.Contains(device, ":")
}
