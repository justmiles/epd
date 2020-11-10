package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/justmiles/epd/lib/dashboard"
	"github.com/spf13/cobra"
)

var (
	displayImage string
	previewImage bool
)

func init() {
	log.SetFlags(0)
	rootCmd.AddCommand(displayImageCmd)

	displayImageCmd.PersistentFlags().StringVarP(&displayImage, "image", "i", "", "image to display")
	displayImageCmd.PersistentFlags().BoolVar(&previewImage, "preview", false, "preview the image instead of updating the display")
}

var displayImageCmd = &cobra.Command{
	Use:   "display-image",
	Short: "Display an image on your EPD",
	Run: func(cmd *cobra.Command, args []string) {

		d, err := dashboard.NewDashboard(device)
		if err != nil {
			panic(err)
		}

		// If this is a URL, let's download it
		if isValidURL(displayImage) {
			tmpfile, err := ioutil.TempFile("", "epd-image")
			if err != nil {
				panic(err)
			}
			defer os.Remove(tmpfile.Name()) // clean up

			err = DownloadFile(displayImage, tmpfile.Name())
			if err != nil {
				panic(err)
			}
			displayImage = tmpfile.Name()
		}

		err = d.DisplayImage(displayImage)
		if err != nil {
			panic(err)
		}

	},
}

// DownloadFile ..
func DownloadFile(fullURLFIle, localFilePath string) (err error) {

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
	defer resp.Body.Close()
	if err != nil {
		return fmt.Errorf("Error downloading logo:\n %s", err)
	}

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
