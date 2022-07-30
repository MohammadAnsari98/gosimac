package unsplash

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/pterm/pterm"
)

// ErrRequestFailed indicates a general error in service request.
var ErrRequestFailed = errors.New("request failed")

// Unsplash image provider.
type Unsplash struct {
	N           int
	Query       string
	Orientation string
	Path        string
	Prefix      string
	Client      *resty.Client
}

func New(count int, query string, orientation string, token string, path string) *Unsplash {
	return &Unsplash{
		N:           count,
		Query:       query,
		Orientation: orientation,
		Path:        path,
		Prefix:      "unsplash",
		Client: resty.New().
			SetBaseURL("https://api.unsplash.com").
			SetHeader("Accept-Version", "v1").
			SetHeader("Authorization", fmt.Sprintf("Client-ID %s", token)),
	}
}

// gather images urls from unplash based on given critarias.
func (u *Unsplash) gather() ([]Image, error) {
	var images []Image

	resp, err := u.Client.R().
		SetResult(&images).
		SetQueryParam("count", strconv.Itoa(u.N)).
		SetQueryParam("orientation", u.Orientation).
		SetQueryParam("query", u.Query).
		Get("/photos/random")
	if err != nil {
		return nil, fmt.Errorf("network failure: %w", err)
	}

	if resp.IsError() {
		pterm.Error.Printf("unplash response code is %d: %s", resp.StatusCode(), resp.String())

		return nil, ErrRequestFailed
	}

	return images, nil
}

// Fetch images from unsplash based on given critarias.
func (u *Unsplash) Fetch() error {
	images, err := u.gather()
	if err != nil {
		return fmt.Errorf("gatering information from unplash failed %w", err)
	}

	// unplash rate limiter is sensivite we reduce the number of goroutines.
	for _, image := range images {
		pterm.Info.Printf("Getting %s (%s)\n", image.ID, image.Description)

		resp, err := resty.New().R().SetDoNotParseResponse(true).Get(image.URLs.Full)
		if err != nil {
			return fmt.Errorf("network failure: %w", err)
		}

		if resp.IsError() {
			pterm.Error.Printf("unplash response code is %d: %s", resp.StatusCode(), resp.String())

			return ErrRequestFailed
		}

		pterm.Success.Printf("%s was gotten\n", image.ID)

		go u.Store(image.ID, resp.RawBody())
	}

	return nil
}

func (u *Unsplash) Store(name string, content io.ReadCloser) {
	path := path.Join(
		u.Path,
		fmt.Sprintf("%s-%s", u.Prefix, name),
	)

	if _, err := os.Stat(path); err == nil {
		pterm.Warning.Printf("%s is already exists\n", path)

		return
	}

	file, err := os.Create(path)
	if err != nil {
		pterm.Error.Printf("os.Create: %v\n", err)

		return
	}

	bytes, err := io.Copy(file, content)
	if err != nil {
		pterm.Error.Printf("io.Copy (%d bytes): %v\n", bytes, err)
	}

	if err := file.Close(); err != nil {
		pterm.Error.Printf("(*os.File).Close: %v", err)
	}

	if err := content.Close(); err != nil {
		pterm.Error.Printf("(*io.ReadCloser).Close: %v", err)
	}
}
