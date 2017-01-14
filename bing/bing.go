// Package bing provides a simple way to access bing API.
package bing

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/franela/goreq"
	"github.com/golang/glog"
)

var w sync.WaitGroup

func getBingImage(path string, image Image) {
	fmt.Printf("Getting %s\n", image.StartDate)

	defer w.Done()

	if _, err := os.Stat(fmt.Sprintf("%s/%s.jpg", path, image.FullStartDate)); err == nil {
		fmt.Printf("%s is already exists\n", image.StartDate)
		return
	}

	resp, err := goreq.Request{
		Uri: fmt.Sprintf("http://www.bing.com/%s", image.URL),
	}.Do()
	if err != nil {
		glog.Errorf("net/http: %v", err)
		return
	}

	defer resp.Body.Close()

	destFile, err := os.Create(fmt.Sprintf("%s/%s.jpg", path, image.FullStartDate))
	if err != nil {
		glog.Errorf("OS: %v\n", err)
		return
	}

	defer destFile.Close()

	io.Copy(destFile, resp.Body)

	fmt.Printf("%s was gotten\n", image.StartDate)
}

// GetBingDesktop function gets `n` since `idx` from bing and store them in `path`.
func GetBingDesktop(path string, idx int, n int) error {
	goreq.SetConnectTimeout(1 * time.Minute)
	// Create HTTP GET request
	resp, err := goreq.Request{
		Uri: "http://www.bing.com/HPImageArchive.aspx",
		QueryString: Request{
			Format: "js",
			Index:  idx,
			Number: n,
			Mkt:    "en-US",
		},
		UserAgent: "GoSiMac",
	}.Do()
	if err != nil {
		glog.Errorf("net/http: %v\n", err)
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Errorf("io/ioutil: %v", err)
	}
	var bingResp Response
	json.Unmarshal(body, &bingResp)

	// Create spreate thread for each image
	for _, image := range bingResp.Images {
		w.Add(1)
		go getBingImage(path, image)
	}

	// Waiting for getting all the images
	w.Wait()

	return nil
}
