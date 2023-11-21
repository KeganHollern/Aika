package web

import (
	"fmt"

	"github.com/gocolly/colly"
)

func TenorToGif(url string) (string, error) {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36"),
	)
	rawUrl := ""
	c.OnHTML("div.Gif:nth-child(13) > img:nth-child(1)", func(e *colly.HTMLElement) {
		rawUrl = e.Attr("src")
	})

	err := c.Visit(url)
	if err != nil {
		return url, fmt.Errorf("failed to visit site; %w", err)
	}
	c.Wait()

	return rawUrl, nil
}
