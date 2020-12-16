package dashboard

// EPDOptions tells the dashboard how to connect to an Electronic Paper Display
type EPDOptions struct {
	Device string
}

// WithEPD creates a dashboard using these Electronic Paper Display options
func WithEPD(device string) Options {
	return func(cd *Dashboard) {
		cd.Device = device
	}
}
