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

	"github.com/PuerkitoBio/goquery"
)

var mfHTTPStatus = map[int]string{
	200: "OK",
	301: "Moved Permanently",
	302: "Found",
	400: "Bad Request",
	403: "Forbidden",
	404: "Not Found",
	429: "Too Many Requests",
	500: "Internal Server Error",
	503: "Service Unavailable",
}

var mfClient = &http.Client{
	Timeout: 15 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return http.ErrUseLastResponse
		}
		return nil
	},
}

var reDirect = regexp.MustCompile(`(?i)https?://download\d+\.mediafire\.com/[^\s"'<>]+`)
var reTitle  = regexp.MustCompile(`(?i)^(.*?)\s*-\s*MediaFire`)
var reDlText = regexp.MustCompile(`(?i)download\s*\(.*?\)`)

// ==== Output ====

type mfMeta struct {
	Creator  string `json:"creator"`
	Status   int    `json:"status"`
	Response string `json:"response"`
}

type mfFile struct {
	Filename    *string `json:"filename"`
	Filesize    *string `json:"filesize"`
	Uploaded    *string `json:"uploaded"`
	Filetype    *string `json:"filetype"`
	DownloadURL *string `json:"download_url"`
}

type mfPayload struct {
	Data []mfFile `json:"data"`
}

func mfPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func mfStatusName(code int) string {
	if s, ok := mfHTTPStatus[code]; ok {
		return s
	}
	return "Unknown"
}

// ==== Helpers ====

// Setara jQuery :contains("text") pada elemen terpilih
func containsText(sel *goquery.Selection, needle string) bool {
	return strings.Contains(strings.ToLower(sel.Text()), strings.ToLower(needle))
}

// ==== Main ====

func mediafireDl(rawURL string) (mfMeta, mfPayload) {
	if rawURL == "" || !strings.Contains(rawURL, "mediafire.com") {
		return mfMeta{Creator: "Bran", Status: 400, Response: "Bad Request"},
			mfPayload{Data: []mfFile{}}
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return mfMeta{Creator: "Bran", Status: 500, Response: "Unknown"}, mfPayload{Data: []mfFile{}}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	res, err := mfClient.Do(req)
	if err != nil {
		return mfMeta{Creator: "Bran", Status: 500, Response: "Unknown"}, mfPayload{Data: []mfFile{}}
	}
	defer res.Body.Close()

	status := res.StatusCode
	body, _ := io.ReadAll(res.Body)
	html := string(body)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return mfMeta{Creator: "Bran", Status: 500, Response: "Unknown"}, mfPayload{Data: []mfFile{}}
	}

	// --- Direct download link ---
	downloadURL := ""
	if v, ok := doc.Find("#downloadButton").Attr("href"); ok {
		downloadURL = v
	}
	if downloadURL == "" {
		doc.Find(`a[href*="download"]`).EachWithBreak(func(_ int, s *goquery.Selection) bool {
			href, _ := s.Attr("href")
			if strings.Contains(href, "download") && strings.Contains(href, "mediafire.com") {
				downloadURL = href
				return false // break
			}
			return true
		})
	}
	if downloadURL == "" {
		if m := reDirect.FindString(html); m != "" {
			downloadURL = m
		}
	}

	// --- Filename ---
	filename := strings.TrimSpace(doc.Find(".filename").First().Text())
	if filename == "" {
		filename = strings.TrimSpace(doc.Find("div.filename").First().Text())
	}
	if filename == "" {
		if t, ok := doc.Find(".dl-btn-label").Attr("title"); ok {
			filename = t
		}
	}
	if filename == "" {
		if m := reTitle.FindStringSubmatch(doc.Find("title").Text()); len(m) > 1 {
			filename = strings.TrimSpace(m[1])
		}
	}

	// --- Filesize ---
	filesize := ""
	doc.Find(".details li").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		if containsText(s, "File size") {
			filesize = cleanLabel(s.Text(), "File size:")
			return false
		}
		return true
	})
	if filesize == "" {
		filesize = cleanLabel(doc.Find("ul.details > li").Eq(0).Text(), "File size:")
	}

	// --- Uploaded ---
	uploaded := ""
	doc.Find(".details li").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		if containsText(s, "Uploaded") {
			uploaded = cleanLabel(s.Text(), "Uploaded:")
			return false
		}
		return true
	})
	if uploaded == "" {
		uploaded = cleanLabel(doc.Find("ul.details > li").Eq(1).Text(), "Uploaded:")
	}

	// --- Filetype ---
	filetype := strings.TrimSpace(doc.Find(".filetype").First().Text())
	if filetype == "" {
		filetype = strings.TrimSpace(doc.Find("div.filetype").First().Text())
	}

	// Cleanup filename
	if strings.Contains(strings.ToLower(filename), "download") {
		filename = strings.TrimSpace(reDlText.ReplaceAllString(filename, ""))
	}

	result := mfFile{
		Filename:    mfPtr(filename),
		Filesize:    mfPtr(filesize),
		Uploaded:    mfPtr(uploaded),
		Filetype:    mfPtr(filetype),
		DownloadURL: mfPtr(downloadURL),
	}

	return mfMeta{Creator: "Bran", Status: status, Response: mfStatusName(status)},
		mfPayload{Data: []mfFile{result}}
}

// setara .text().replace(/Label:/i, '').trim()
func cleanLabel(text, label string) string {
	// case-insensitive replace sekali
	idx := strings.Index(strings.ToLower(text), strings.ToLower(label))
	if idx >= 0 {
		text = text[:idx] + text[idx+len(label):]
	}
	return strings.TrimSpace(text)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run mediafire.go <mediafire-url>")
		return
	}
	meta, payload := mediafireDl(os.Args[1])
	out, _ := json.MarshalIndent([]any{meta, payload}, "", "  ")
	fmt.Println(string(out))
}
