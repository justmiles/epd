# Electronic Paper Device

Interact with a supported Electronic Paper Display — locally or remotely via gRPC.

## Usage

```shell
Interact with a supported Electronic Paper Display

Usage:
  epd [flags]
  epd [command]

Available Commands:
  clear             Clear the EPD to white
  display-image     Display an image on your EPD
  display-text      Display text on your EPD
  help              Help about any command
  refresh-dashboard Update your display with a custom dashboard
  serve             Run as a daemon, exposing the EPD over gRPC

Flags:
      --debug           enable debug logging
  -d, --device string   your supported EPD device type (eh, I only have epd7in5v2 right now.) (default "epd7in5v2")
  -h, --help            help for epd
  -i, --initialize      initialize (wake) the device before updating it. Required if in sleep mode
  -s, --sleep           set the device to sleep mode after updating display
      --version         version for epd

Use "epd [command] --help" for more information about a command.
```

## Daemon Mode (Remote Display)

Run the EPD as a daemon on the device connected to the display (e.g. your Raspberry Pi):

```bash
epd serve --port 50051
```

From any other machine, use the same commands with `--device host:port` to push content remotely:

```bash
# Display an image remotely
epd display-image --device pi.local:50051 dashboard-image.png

# Display text remotely
epd display-text --device pi.local:50051 "Hello World"

# Clear the display remotely
epd clear --device pi.local:50051
```

When `--device` contains a `host:port` (e.g. `pi.local:50051`), commands automatically connect via gRPC. Otherwise, they operate on local hardware as before.

## Generate a dashboard

You'll need to grab an [openweathermap](https://openweathermap.org/) API key and setup TaskWarrior. Once this in is place, you can use the `epd refresh-dashboard` command to display a generic dashboard. I schedule this in cron on my raspberry PI to update every 30 minutes.

example:

```bash
epd refresh-dashboard --dstask --weather-api-key XXXXXXXXXXXX
```

You can also generate the dashboard locally and push it to a remote display:

```bash
epd refresh-dashboard --dstask --weather-api-key XXXXXXXXXXXX --device pi.local:50051
```

![dashboard-image](https://justmiles.keybase.pub/assets/github.com/justmiles/epd/dashboard-image.png)

## Supported Displays

| Model                                                           | Tested On         |
| --------------------------------------------------------------- | ----------------- |
| [epd7in5v2](https://www.waveshare.com/wiki/7.5inch_e-Paper_HAT) | Raspberry PI Zero |
