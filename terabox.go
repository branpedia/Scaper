package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type Download struct {
	Type     string `json:"type"`
	Filename string `json:"filename"`
	Size     string `json:"size"`
	Quality  string `json:"quality"`
	URL      string `json:"url"`
}

type Result struct {
	Title      string     `json:"title"`
	Thumbnail  string     `json:"thumbnail"`
	TotalFiles int        `json:"totalFiles"`
	Downloads  []Download `json:"downloads"`
}

type Resp struct {
	Status  bool    `json:"status"`
	Result  *Result `json:"result,omitempty"`
	Message string  `json:"message,omitempty"`
}

// ==== payload dari API sechno ====

type apiFile struct {
	IsDir           bool   `json:"isDir"`
	Filename        string `json:"filename"`
	Size            int64  `json:"size"`
	SizeFormatted   string `json:"sizeFormatted"`
	DownloadURL     string `json:"downloadUrl"`
	StreamURL       string `json:"streamUrl"`
	ThumbURL        string `json:"thumbUrl"`
}

type apiResp struct {
	Success bool      `json:"success"`
	Error   string    `json:"error"`
	Title   string    `json:"title"`
	Files   []apiFile `json:"files"`
}

var reURL = regexp.MustCompile(`https?://[^\s]+`)

func extractCleanURL(text string) string {
	if text == "" {
		return ""
	}
	clean := text
	if m := reURL.FindString(text); m != "" {
		clean = m
	} else {
		clean = strings.TrimSpace(text)
	}
	if !strings.HasPrefix(clean, "http://") && !strings.HasPrefix(clean, "https://") {
		clean = "https://" + clean
	}
	return clean
}

func scrape(rawURL string) Resp {
	if rawURL == "" {
		return Resp{Message: "Invalid URL."}
	}
	cleanURL := extractCleanURL(rawURL)

	body, _ := json.Marshal(map[string]string{"url": cleanURL})

	req, err := http.NewRequest("POST", "https://sechno.com/api/terabox", bytes.NewReader(body))
	if err != nil {
		return Resp{Message: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://sechno.com/tools/terabox-downloader")

	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return Resp{Message: err.Error()}
	}
	defer res.Body.Close()

	raw, _ := io.ReadAll(res.Body)

	var data apiResp
	if err := json.Unmarshal(raw, &data); err != nil {
		return Resp{Message: err.Error()}
	}

	if !data.Success {
		msg := data.Error
		if msg == "" {
			msg = "Failed to resolve TeraBox link."
		}
		return Resp{Message: msg}
	}

	title := data.Title
	if title == "" {
		title = "TeraBox Shared Files"
	}
	if len(data.Files) == 0 {
		return Resp{Message: "No files found in TeraBox share."}
	}

	var downloads []Download
	for _, f := range data.Files {
		if f.IsDir {
			continue
		}
		size := f.SizeFormatted
		if size == "" {
			size = fmt.Sprintf("%d B", f.Size)
		}
		if f.DownloadURL != "" {
			downloads = append(downloads, Download{
				Type:     "file",
				Filename: f.Filename,
				Size:     size,
				Quality:  "Direct Download",
				URL:      f.DownloadURL,
			})
		}
		if f.StreamURL != "" {
			downloads = append(downloads, Download{
				Type:     "video",
				Filename: f.Filename,
				Size:     size,
				Quality:  "Fast Stream (m3u8)",
				URL:      f.StreamURL,
			})
		}
	}

	firstThumb := ""
	for _, f := range data.Files {
		if f.ThumbURL != "" {
			firstThumb = f.ThumbURL
			break
		}
	}

	return Resp{
		Status: true,
		Result: &Result{
			Title:      title,
			Thumbnail:  firstThumb,
			TotalFiles: len(data.Files),
			Downloads:  downloads,
		},
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run terabox.go <terabox-url>")
		return
	}
	out, _ := json.MarshalIndent(scrape(os.Args[1]), "", "  ")
	fmt.Println(string(out))
}
