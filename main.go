package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/fujiwara/shapeio"
	"github.com/gosimple/slug"
	"github.com/mmcdole/gofeed"
)

// TODO: read in a list of podcasts from an opml file
// TODO: update rss feed with local links for audio files before writing to disk
// TODO: download item images, update rss feed with local links for them
// TODO: generate html representation of the rss feed
// TODO: flag for basedir to archive podcast to

const rateLimit = 1024 * 1024 * 10 // 10MB/s

func main() {
	parser := gofeed.NewParser()
	for _, feedURL := range os.Args[1:] {
		feed, err := parser.ParseURL(feedURL)
		if err != nil {
			log.Print(err)
			continue
		}
		log.Printf("Processing %v (%v)\n", feed.Title, feedURL)

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
			itemPath := filepath.Join(itemDir, filepath.Base(enclosureURL.Path))
			// TODO: continue if we have the full enclosure already

			resp, err := http.Get(enclosure.URL)
			if err != nil {
				log.Print(err)
				continue
			}
			defer resp.Body.Close()
			// TODO: verify response content-length matches enclosure length

			itemFile, err := os.OpenFile(itemPath, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				log.Print(err)
				continue
			}
			defer itemFile.Close()

			reader := shapeio.NewReader(resp.Body)
			reader.SetRateLimit(rateLimit)
			_, err = io.Copy(itemFile, reader)
			if err != nil {
				log.Print(err)
				// TODO: remove partially downloaded file
				continue
			}
			// TODO: verify io.Copy return value matches enclosure length
		}
	}
}
