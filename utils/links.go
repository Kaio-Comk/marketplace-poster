package utils

import (
	"strings"
	"time"

	"github.com/go-rod/rod"
)

// LatestListingLink opens "your listings" and returns the first listing link and its numeric id.
// Facebook lists the newest ad first. Empty strings mean it could not be read.
func LatestListingLink(page *rod.Page) (link, id string) {
	if err := page.Navigate("https://www.facebook.com/marketplace/you/selling"); err != nil {
		return "", ""
	}
	time.Sleep(8 * time.Second)

	res, err := page.Eval(`() => { const a = document.querySelector('a[href*="/marketplace/item/"]'); return a ? a.href : "" }`)
	if err != nil {
		return "", ""
	}

	link = res.Value.Str()
	_, rest, found := strings.Cut(link, "/marketplace/item/")
	if !found {
		return "", ""
	}
	id, _, _ = strings.Cut(rest, "/")
	id, _, _ = strings.Cut(id, "?")

	return strings.Split(link, "?")[0], id
}
