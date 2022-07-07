package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/fujiwara/shapeio"
	"github.com/gosimple/slug"
	"github.com/mmcdole/gofeed"
)

// TODO: read in a list of podcasts from an opml file
// TODO: update rss feed with local links for audio files before writing to disk
// TODO: download item images, update rss feed with local links for them
// TODO: generate html representation of the rss feed
// TODO: flag for basedir to archive podcast(s) to

const rateLimit = 1024 * 1024 * 5 // 5 Mbps

func main() {
	parser := gofeed.NewParser()
	for _, feedURL := range os.Args[1:] {
		feed, err := parser.ParseURL(feedURL)
		if err != nil {
			log.Print(err)
			continue
		}
		log.Printf("INFO - Processing %v (%v)\n", feed.Title, feedURL)

		baseDir, err := os.Getwd()
		if err != nil {
			log.Print(err)
			continue
		}

		feedDir := filepath.Join(baseDir, feed.Title)
		err = os.Mkdir(feedDir, 0755)
		if err != nil && !os.IsExist(err) {
			log.Print(err)
			continue
		}

		// TODO: write rss feed out as index.rss

		for _, item := range feed.Items {
			if len(item.Enclosures) > 1 {
				log.Printf("%v - %v has more than one enclosure, skipping\n",
					feed.Title, item.Title)
				continue
			}
			enclosure := item.Enclosures[0]
			enclosureLength, err := strconv.ParseInt(enclosure.Length, 10, 64)
			if err != nil {
				log.Print(err)
				continue
			}

			itemDir := filepath.Join(feedDir, slug.Make(item.GUID))
			err = os.Mkdir(itemDir, 0755)
			if err != nil && !os.IsExist(err) {
				log.Print(err)
				continue
			}

			enclosureURL, err := url.Parse(enclosure.URL)
			if err != nil {
				log.Print(err)
				continue
			}
			enclosurePath := filepath.Join(itemDir, filepath.Base(enclosureURL.Path))
			if stat, err := os.Stat(enclosurePath); err == nil {
				resp, err := http.Head(enclosure.URL)
				if err != nil {
					log.Printf("ERROR: %v %v\n", enclosure.URL, err)
					continue
				}
				if resp.ContentLength == stat.Size() {
					log.Printf("INFO: %v already downloaded, skipping\n", enclosure.URL)
					continue
				} else {
					log.Printf("WARN: %v partially downloaded, overwriting\n", enclosure.URL)
				}
			}

			resp, err := http.Get(enclosure.URL)
			if err != nil {
				log.Print(err)
				continue
			}
			defer resp.Body.Close()

			if resp.ContentLength != enclosureLength {
				log.Printf("WARN: %v enclosure length %#v, content length %#v\n",
					enclosure.URL, enclosureLength, resp.ContentLength)
			}

			enclosureFile, err := os.OpenFile(enclosurePath, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				log.Print(err)
				continue
			}
			defer enclosureFile.Close()

			reader := shapeio.NewReader(resp.Body)
			reader.SetRateLimit(rateLimit)
			copyLength, err := io.Copy(enclosureFile, reader)
			if err != nil {
				log.Print(err)
				os.Remove(enclosurePath)
				continue
			}
			if copyLength != resp.ContentLength {
				log.Printf("WARN: %v download length %#v, content length %#v\n",
					enclosure.URL, copyLength, resp.ContentLength)
			}
		}
	}
}
