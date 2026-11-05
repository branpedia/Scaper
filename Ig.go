package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Download struct {
	Type      string `json:"type"`
	URL       string `json:"url"`
	Thumbnail string `json:"thumbnail"`
}

type Result struct {
	Title     string     `json:"title"`
	Thumbnail string     `json:"thumbnail"`
	Downloads []Download `json:"downloads"`
}

type Resp struct {
	Status  bool    `json:"status"`
	Result  *Result `json:"result,omitempty"`
	Message string  `json:"message,omitempty"`
}

var reStrip = regexp.MustCompile(`<[^>]+>`)
var reToken = regexp.MustCompile(`token=([A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+)`)
var reVid   = regexp.MustCompile(`(?i)\.(mp4|mov|webm|mkv)(\?|"|$)`)
var reImg   = regexp.MustCompile(`(?i)\.(jpe?g|png|webp)(\?|")`)
var reHref  = regexp.MustCompile(`(?i)\.(mp4|mov|webm)(\?|$)`)

func checkIsVideo(href, text, title string) bool {
	upper := strings.ToUpper(text + " " + title)
	if strings.Contains(upper, "VIDEO") || strings.Contains(upper, "MP4") || strings.Contains(upper, "REEL") {
		return true
	}
	if strings.Contains(upper, "PHOTO") || strings.Contains(upper, "IMAGE") {
		return false
	}
	if m := reToken.FindStringSubmatch(href); m != nil {
		parts := strings.Split(m[1], ".")
		if len(parts) > 1 {
			p := strings.NewReplacer("-", "+", "_", "/").Replace(parts[1])
			for len(p)%4 != 0 {
				p += "="
			}
			if dec, err := base64.StdEncoding.DecodeString(p); err == nil {
				s := string(dec)
				if reVid.MatchString(s) {
					return true
				}
				if reImg.MatchString(s) {
					return false
				}
			}
		}
	}
	return reHref.MatchString(href)
}

func scrape(rawURL string) Resp {
	clean := strings.Split(rawURL, "?")[0]

	form := url.Values{}
	form.Set("q", clean)
	form.Set("vt", "reel")
	form.Set("t", "media")
	form.Set("lang", "en")
	form.Set("v", "v2")

	req, err := http.NewRequest("POST", "https://indown.net/api/ajaxSearch", strings.NewReader(form.Encode()))
	if err != nil {
		return Resp{Message: err.Error()}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", "https://indown.net")
	req.Header.Set("Referer", "https://indown.net/")

	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return Resp{Message: err.Error()}
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	var j struct {
		Status string `json:"status"`
		Data   string `json:"data"`
		Mess   string `json:"mess"`
	}
	if err := json.Unmarshal(body, &j); err != nil {
		return Resp{Message: err.Error()}
	}

	if j.Status != "ok" || j.Data == "" {
		msg := strings.TrimSpace(reStrip.ReplaceAllString(j.Mess, ""))
		if msg == "" {
			msg = "Failed to process Instagram URL on indown.net"
		}
		return Resp{Message: msg}
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(j.Data))
	if err != nil {
		return Resp{Message: err.Error()}
	}

	var downloads []Download
	thumbnail := ""
	seen := map[string]bool{}

	doc.Find(".download-items").Each(func(_ int, el *goquery.Selection) {
		thumb, _ := el.Find("img").Attr("src")
		if thumb != "" && thumbnail == "" {
			thumbnail = thumb
		}
		el.Find(`a[href*='download'], a.abutton, a.btn, a[href*='snapcdn'], a[href*='rapidcdn'], a[href*='cdninstagram']`).
			Each(func(_ int, a *goquery.Selection) {
				href, _ := a.Attr("href")
				if href == "" || !strings.HasPrefix(href, "http") || seen[href] {
					return
				}
				text := strings.TrimSpace(a.Text())
				title, _ := a.Attr("title")

				t := "image"
				if checkIsVideo(href, text, title) {
					t = "video"
				}
				tb := thumb
				if tb == "" {
					tb = href
				}
				seen[href] = true
				downloads = append(downloads, Download{Type: t, URL: href, Thumbnail: tb})
			})
	})

	if len(downloads) == 0 {
		return Resp{Message: "No media links found. The post might be private or unavailable."}
	}

	thumb := thumbnail
	if thumb == "" {
		thumb = downloads[0].Thumbnail
	}

	return Resp{
		Status: true,
		Result: &Result{
			Title:     "Instagram Media",
			Thumbnail: thumb,
			Downloads: downloads,
		},
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run main.go <instagram-url>")
		return
	}
	out, _ := json.MarshalIndent(scrape(os.Args[1]), "", "  ")
	fmt.Println(string(out))
}
