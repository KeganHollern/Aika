package web_test

import (
	"aika/actions/web"
	"testing"

	"github.com/stretchr/testify/assert"
)

var tenorURLs = map[string]string{
	`https://tenor.com/view/hhgf-gif-25031041`:                 "https://media.tenor.com/2w1XsfvQD5kAAAAC/hhgf.gif",
	`https://tenor.com/view/gold-star-succession-gif-27712928`: "https://media.tenor.com/AfEYowNklBkAAAAC/gold-star.gif",
}

func TestTenor(t *testing.T) {
	for in, out := range tenorURLs {
		url, err := web.TenorToGif(in)
		assert.NoError(t, err)
		assert.Equal(t, out, url)
	}
}
