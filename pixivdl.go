package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const chromeUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

type Download struct {
	Type    string `json:"type"`
	Quality string `json:"quality"`
	URL     string `json:"url"`
}

type Author struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type Result struct {
	Title     string     `json:"title"`
	Thumbnail string     `json:"thumbnail"`
	Type      string     `json:"type"`
	Author    Author     `json:"author"`
	Downloads []Download `json:"downloads"`
}

type Resp struct {
	Status  bool    `json:"status"`
	Result  *Result `json:"result,omitempty"`
	Message string  `json:"message,omitempty"`
}

// ==== Illust data (gabungan AJAX + preload JSON) ====

type illustURLs struct {
	Regular  string `json:"regular"`
	Original string `json:"original"`
}

type illustBody struct {
	IllustType  json.RawMessage `json:"illustType"`
	Type        string          `json:"type"`
	Title       string          `json:"title"`
	IllustTitle string          `json:"illustTitle"`
	UserName    string          `json:"userName"`
	UserAccount string          `json:"userAccount"`
	UserID      string          `json:"userId"`
	PageCount   int             `json:"pageCount"`
	URLs        illustURLs      `json:"urls"`
}

var (
	reArtwork  = regexp.MustCompile(`artworks/(\d+)`)
	reIllustID = regexp.MustCompile(`illust_id=(\d+)`)
	rePreload1 = regexp.MustCompile(`(?i)id="meta-preload-data"\s+content='([^']+)'`)
	rePreload2 = regexp.MustCompile(`(?i)id="meta-preload-data"\s+content="([^"]+)"`)
	reUserID   = regexp.MustCompile(`users/(\d+)`)
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// ==== Helper fetch ====

func httpGet(url string, extraHeaders map[string]string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", chromeUA)
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

// ==== Main scrape ====

func scrapePixiv(rawURL string) Resp {
	m := reArtwork.FindStringSubmatch(rawURL)
	if m == nil {
		m = reIllustID.FindStringSubmatch(rawURL)
	}
	if m == nil {
		return Resp{Message: "Invalid Pixiv URL."}
	}
	illustID := m[1]

	var data *illustBody

	// 1. Official AJAX
	if body, err := httpGet(
		fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s?lang=en", illustID),
		map[string]string{"Referer": "https://www.pixiv.net/"},
	); err == nil {
		var aj struct {
			Error bool         `json:"error"`
			Body  *illustBody  `json:"body"`
		}
		if json.Unmarshal(body, &aj) == nil && !aj.Error && aj.Body != nil {
			data = aj.Body
		}
	}

	// 2. HTML meta-preload-data fallback
	if data == nil {
		if body, err := httpGet(
			fmt.Sprintf("https://www.pixiv.net/en/artworks/%s", illustID),
			map[string]string{"Accept-Language": "en-US,en;q=0.9"},
		); err == nil {
			html := string(body)
			pm := rePreload1.FindStringSubmatch(html)
			if pm == nil {
				pm = rePreload2.FindStringSubmatch(html)
			}
			if pm != nil {
				raw := strings.NewReplacer("&quot;", `"`, "&amp;", "&").Replace(pm[1])
				var preload struct {
					Illust map[string]*illustBody `json:"illust"`
				}
				if json.Unmarshal([]byte(raw), &preload) == nil {
					data = preload.Illust[illustID]
				}
			}
		}
	}

	// 3. oEmbed fallback
	if data == nil {
		if body, err := httpGet(
			fmt.Sprintf("https://embed.pixiv.net/oembed.php?url=https://www.pixiv.net/artworks/%s", illustID),
			nil,
		); err == nil {
			var d struct {
				Title      string `json:"title"`
				AuthorName string `json:"author_name"`
				AuthorURL  string `json:"author_url"`
				Thumbnail  string `json:"thumbnail_url"`
				WorkType   string `json:"work_type"`
			}
			if json.Unmarshal(body, &d) == nil && d.Title != "" {
				authorID := ""
				if am := reUserID.FindStringSubmatch(d.AuthorURL); am != nil {
					authorID = am[1]
				}
				thumb := d.Thumbnail
				if thumb == "" {
					thumb = fmt.Sprintf("https://pixiv.re/%s.jpg", illustID)
				}
				typ := "image"
				if d.WorkType == "ugoira" {
					typ = "video"
				}
				authorName := d.AuthorName
				if authorName == "" {
					authorName = "Pixiv Artist"
				}
				return Resp{Status: true, Result: &Result{
					Title:     fmt.Sprintf("%s by %s", d.Title, authorName),
					Thumbnail: thumb,
					Type:      typ,
					Author:    Author{Name: authorName, ID: authorID},
					Downloads: []Download{{
						Type:    "image",
						Quality: "Original",
						URL:     fmt.Sprintf("https://pixiv.re/%s.jpg", illustID),
					}},
				}}
			}
		}
	}

	if data == nil {
		return Resp{Message: "Could not fetch Pixiv artwork data."}
	}

	// ==== Detect ugoira ====
	isUgoira := data.Type == "ugoira" ||
		strings.Trim(string(data.IllustType), `"`) == "2"

	// ==== Title ====
	baseTitle := data.Title
	if baseTitle == "" {
		baseTitle = data.IllustTitle
	}
	authorName := data.UserName
	if authorName == "" {
		authorName = data.UserAccount
	}
	if authorName == "" {
		authorName = "Artist"
	}
	title := "Pixiv Artwork"
	if baseTitle != "" {
		title = fmt.Sprintf("%s by %s", baseTitle, authorName)
	}

	var downloads []Download

	if isUgoira {
		downloads = append(downloads,
			Download{Type: "video", Quality: "MP4 Animation", URL: fmt.Sprintf("https://ugoira.com/api/mp4/%s", illustID)},
			Download{Type: "image", Quality: "GIF Animation", URL: fmt.Sprintf("https://pixiv.re/%s.gif", illustID)},
		)
	} else {
		pageCount := data.PageCount
		if pageCount < 1 {
			pageCount = 1
		}
		if data.URLs.Original != "" {
			for i := 0; i < pageCount; i++ {
				quality := "Original"
				if pageCount > 1 {
					quality = fmt.Sprintf("Page %d", i+1)
				}
				pageURL := strings.Replace(data.URLs.Original, "_p0", fmt.Sprintf("_p%d", i), 1)
				pageURL = strings.Replace(pageURL, "i.pximg.net", "i.pixiv.re", 1)
				downloads = append(downloads, Download{Type: "image", Quality: quality, URL: pageURL})
			}
		} else {
			downloads = append(downloads, Download{
				Type:    "image",
				Quality: "Original",
				URL:     fmt.Sprintf("https://pixiv.re/%s.jpg", illustID),
			})
		}
	}

	// ==== Thumbnail ====
	var thumb string
	if isUgoira {
		thumb = fmt.Sprintf("https://pixiv.re/%s.gif", illustID)
	} else if data.URLs.Regular != "" {
		thumb = strings.Replace(data.URLs.Regular, "i.pximg.net", "i.pixiv.re", 1)
	} else if data.URLs.Original != "" {
		thumb = strings.Replace(data.URLs.Original, "i.pximg.net", "i.pixiv.re", 1)
	} else {
		thumb = fmt.Sprintf("https://pixiv.re/%s.jpg", illustID)
	}

	typ := "image"
	if isUgoira {
		typ = "video"
	}

	return Resp{Status: true, Result: &Result{
		Title:     title,
		Thumbnail: thumb,
		Type:      typ,
		Author:    Author{Name: authorName, ID: data.UserID},
		Downloads: downloads,
	}}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run pixiv.go <pixiv-url>")
		return
	}
	out, _ := json.MarshalIndent(scrapePixiv(os.Args[1]), "", "  ")
	fmt.Println(string(out))
}
