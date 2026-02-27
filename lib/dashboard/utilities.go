package dashboard

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/font/gofont/goregular"
)

func convertSVGToImage(stream io.Reader, width, height float64) image.Image {

	icon, errSvg := oksvg.ReadIconStream(stream, oksvg.IgnoreErrorMode)
	if errSvg != nil {
		fmt.Println(errSvg)
	}
	w, h := int(icon.ViewBox.W), int(icon.ViewBox.H)
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	scannerGV := rasterx.NewScannerGV(w, h, img, img.Bounds())
	raster := rasterx.NewDasher(w, h, scannerGV)
	// icon.
	raster.SetColor(color.White)
	// raster.Fi
	icon.SetTarget(0, 0, width, height)
	icon.Draw(raster, 1.0)

	return img
}

// buildCalendarWidget x pixels wide and y pixels tall
func buildCalendarWidget(x, y int, location string) (image.Image, error) {

	//init the loc
	var loc *time.Location
	if location != "" {
		var err error
		loc, err = time.LoadLocation(location)
		if err != nil {
			return nil, fmt.Errorf("Invalid location: %s", err)
		}
	} else {
		loc = time.Now().Location()
	}

	var (
		xWidth, xHeight          = float64(x), float64(y)
		fontSize, widgetLocation float64
		now                      = time.Now().In(loc)
	)

	// Draw background
	dc := gg.NewContext(x, y)
	dc.DrawRectangle(0, 0, xWidth, xHeight)
	dc.SetRGB(0, 0, 0)
	dc.Fill()

	// Set font color
	dc.SetRGB(1, 1, 1)

	// Draw day of the week
	dow := now.Format("Monday")
	fontSize = setDynamicFont(dc, xWidth-(xWidth*.1), xHeight*.2, dow)
	widgetLocation = fontSize/2 + 10
	// fmt.Println(widgetLocation)
	dc.DrawStringAnchored(dow, xWidth/2, widgetLocation, 0.5, 0.5)

	// Draw day of month
	domText := now.Format("02")
	fontSize = setDynamicFont(dc, xWidth-(xWidth*.1), xHeight*.5, domText)
	widgetLocation = (widgetLocation) + fontSize/2 + 10
	// fmt.Println(widgetLocation)

	dc.DrawStringAnchored(domText, xWidth/2, widgetLocation, 0.5, 0.5)

	// Draw month, year
	ymText := now.Format("January 2006")
	fontSize = setDynamicFont(dc, xWidth-(xWidth*.1), xHeight*.3, ymText)
	widgetLocation = (widgetLocation * 1.75) + fontSize/2
	// fmt.Println(widgetLocation)

	dc.DrawStringAnchored(ymText, xWidth/2, widgetLocation, 0.5, 0.5)

	return dc.Image(), nil
}

func fitTextInBox(dc *gg.Context, text string, xx, yy float64, x, y int) error {
	var (
		// maxWidth, maxHeight           float64 = float64(x), float64(y)
		fontSize float64 = 30 // initial font size
		// fontSizeReduction             float64 = 0.95       // reduce the font size by this much until message fits in the display
		// fontSizeMinimum               float64 = 10         // Smallest font size before giving up
		// lineSpacing                   float64 = 1
		// measuredWidth, measuredHeight float64
	)

	setFont(dc, fontSize)

	// // Fit the text in the box!
	// for {
	// 	face := truetype.NewFace(font, &truetype.Options{Size: fontSize})
	// 	dc.SetFontFace(face)

	// 	stringLines := dc.WordWrap(text, maxWidth)

	// 	measuredWidth, measuredHeight = dc.MeasureMultilineString(strings.Join(stringLines, "\n"), lineSpacing)

	// 	// If the message fits within the frame, let's break. Otherwise reduce the font size and try again
	// 	if measuredWidth < maxWidth && measuredHeight <= maxHeight {
	// 		break
	// 	} else {
	// 		fontSize = fontSize * fontSizeReduction
	// 	}

	// 	if fontSize < fontSizeMinimum {
	// 		return fmt.Errorf("unable to fit text on screen: \n %s", text)
	// 	}
	// }

	dc.DrawString(text, xx, yy)
	return nil
}

func drawTextInBox(text string, x, y int) (image.Image, error) {

	// Create new logo context
	dc := gg.NewContext(x, y)

	// Set Background Color
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// Set font color
	dc.SetColor(color.Black)

	dc.Fill()
	dc.SetRGB(0, 0, 0)

	var (
		maxWidth, maxHeight           float64 = float64(x), float64(y)
		fontSize                      float64 = 300  // initial font size
		fontSizeReduction             float64 = 0.95 // reduce the font size by this much until message fits in the display
		fontSizeMinimum               float64 = 10   // Smallest font size before giving up
		lineSpacing                   float64 = 1
		measuredWidth, measuredHeight float64
	)

	// Fit the text in the box!
	for {
		setFont(dc, fontSize)

		stringLines := dc.WordWrap(text, maxWidth)

		measuredWidth, measuredHeight = dc.MeasureMultilineString(strings.Join(stringLines, "\n"), lineSpacing)

		// If the message fits within the frame, let's break. Otherwise reduce the font size and try again
		if measuredWidth < maxWidth && measuredHeight <= maxHeight {
			break
		} else {
			fontSize = fontSize * fontSizeReduction
		}

		if fontSize < fontSizeMinimum {
			return nil, fmt.Errorf("unable to fit text on screen: \n %s", text)
		}
	}

	dc.DrawStringWrapped(text, 0, (maxHeight-measuredHeight)/2-(fontSize/4), 0, 0, maxWidth, lineSpacing, gg.AlignCenter)

	return dc.Image(), nil
}

func setFont(dc *gg.Context, fontSize float64) {
	font, _ := truetype.Parse(goregular.TTF)
	face := truetype.NewFace(font, &truetype.Options{
		Size: fontSize,
	})
	dc.SetFontFace(face)
}

func setDynamicFont(dc *gg.Context, maxWidth, maxHeight float64, s string) float64 {
	fontSize := maxHeight

	for {
		setFont(dc, fontSize)
		width, height := dc.MeasureMultilineString(s, 0)
		if width < maxWidth && height < maxHeight {
			break
		}
		// fmt.Printf("Text Size: %v\n", fontSize)
		fontSize = fontSize * .95
	}

	return fontSize
}

func downloadFile(fullURLFIle, localFilePath string) (err error) {

	// Create blank file
	file, err := os.Create(localFilePath)
	if err != nil {
		return fmt.Errorf("Error creating temporary file:\n %s", err)
	}

	// Put content on file
	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	// Invoke HTTP request
	resp, err := client.Get(fullURLFIle)
	if err != nil {
		return fmt.Errorf("Error downloading logo:\n %s", err)
	}
	defer resp.Body.Close()

	// Pipe data to disk
	_, err = io.Copy(file, resp.Body)
	defer file.Close()
	if err != nil {
		return fmt.Errorf("Error writing logo to disk:\n %s", err)
	}

	return nil
}

// isValidURL determins if a string is an actual URL
func isValidURL(toTest string) bool {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}

	u, err := url.Parse(toTest)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}
