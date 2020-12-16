# Electronic Paper Device

Interact with a supported Electronic Paper Display

## Usage

```shell
Interact with a supported Electronic Paper Display

Usage:
  epd [flags]
  epd [command]

Available Commands:
  display-image     Display an image on your EPD
  display-text      Display text on your EPD
  help              Help about any command
  refresh-dashboard 

Flags:
      --debug           enable debug logging
  -d, --device string   your supported EPD device type (eh, I only have epd7in5v2 right now.) (default "epd7in5v2")
  -h, --help            help for epd
  -i, --initialize      initialize (wake) the device before updating it. Required if in sleep mode
  -s, --sleep           set the device to sleep mode after updating display
      --version         version for epd

Use "epd [command] --help" for more information about a command.
```

## Generate a dashboard

You'll need to grab an [openweathermap](https://openweathermap.org/) API key and setup TaskWarrior. Once this in is place, you can use the `epd refresh-dashboard` command to display a generic dashboard. I schedule this in cron on my raspberry PI to update every 30 minutes.

![dashboard-image](https://justmiles.keybase.pub/assets/github.com/justmiles/epd/dashboard-image.png)

## Supported Displays

| Model                                                           | Tested On         |
| --------------------------------------------------------------- | ----------------- |
| [epd7in5v2](https://www.waveshare.com/wiki/7.5inch_e-Paper_HAT) | Raspberry PI Zero |
