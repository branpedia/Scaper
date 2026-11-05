package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

const stBaseURL = "https://savetiktok.id"

var stHTTPStatus = map[int]string{
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

var stClient = &http.Client{Timeout: 30 * time.Second}

// ==== Output types ====

type stMeta struct {
	Creator  string `json:"creator"`
	Status   int    `json:"status"`
	Response string `json:"response"`
}

type stResult struct {
	Title         *string `json:"title"`
	Author        *string `json:"author"`
	Duration      *string `json:"duration"`
	Cover         *string `json:"cover"`
	MusicTitle    *string `json:"music_title"`
	VideoNowm     *string `json:"video_nowm"`
	Audio         *string `json:"audio"`
	FilenameNowm  *string `json:"filename_nowm"`
	FilenameAudio *string `json:"filename_audio"`
}

type stPayload struct {
	Data []stResult `json:"data"`
}

// ==== API response ====

type stAPIResp struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	Title         string `json:"title"`
	Author        string `json:"author"`
	Duration      string `json:"duration"`
	Cover         string `json:"cover"`
	MusicTitle    string `json:"music_title"`
	VideoNowm     string `json:"video_nowm"`
	Audio         string `json:"audio"`
	FilenameNowm  string `json:"filename_nowm"`
	FilenameAudio string `json:"filename_audio"`
}

func ptrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func statusName(code int) string {
	if s, ok := stHTTPStatus[code]; ok {
		return s
	}
	return "Unknown"
}

// ==== Downloader ====

func tiktokDownloader(rawURL string) (stMeta, stPayload) {
	if rawURL == "" {
		return stMeta{Creator: "Azén", Status: 400, Response: "Bad Request"}, stPayload{Data: []stResult{}}
	}

	// Multipart form
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("url", rawURL)
	w.Close()

	req, err := http.NewRequest("POST", stBaseURL+"/api/downloader.php", &buf)
	if err != nil {
		return stMeta{Creator: "Azén", Status: 500, Response: "Unknown"}, stPayload{Data: []stResult{}}
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Origin", "https://savetiktok.id")
	req.Header.Set("Referer", "https://savetiktok.id/")

	res, err := stClient.Do(req)
	if err != nil {
		return stMeta{Creator: "Azén", Status: 500, Response: "Unknown"}, stPayload{Data: []stResult{}}
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	status := res.StatusCode

	var data stAPIResp
	_ = json.Unmarshal(body, &data)

	if !data.Success {
		msg := data.Message
		if msg == "" {
			msg = "Bad Request"
		}
		return stMeta{Creator: "Azén", Status: 400, Response: msg}, stPayload{Data: []stResult{}}
	}

	result := stResult{
		Title:         ptrOrNil(data.Title),
		Author:        ptrOrNil(data.Author),
		Duration:      ptrOrNil(data.Duration),
		Cover:         ptrOrNil(data.Cover),
		MusicTitle:    ptrOrNil(data.MusicTitle),
		VideoNowm:     ptrOrNil(data.VideoNowm),
		Audio:         ptrOrNil(data.Audio),
		FilenameNowm:  ptrOrNil(data.FilenameNowm),
		FilenameAudio: ptrOrNil(data.FilenameAudio),
	}

	return stMeta{Creator: "Azén", Status: status, Response: statusName(status)},
		stPayload{Data: []stResult{result}}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run savetiktok.go <tiktok-url>")
		return
	}
	meta, payload := tiktokDownloader(os.Args[1])
	out, _ := json.MarshalIndent([]any{meta, payload}, "", "  ")
	fmt.Println(string(out))
}
