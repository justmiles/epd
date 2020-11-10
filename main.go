package main

import (
	"github.com/justmiles/epd/cmd"
)

// version of epd. Overwritten during build
var version = "0.0.0"

func main() {
	cmd.Execute(version)
}
