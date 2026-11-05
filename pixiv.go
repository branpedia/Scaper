package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const baseURL = "https://www.pixiv.net"

var httpStatus = map[int]string{
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

var searchClient = &http.Client{Timeout: 15 * time.Second}

// ==== Output types (mirror struktur array JS kamu) ====

type Meta struct {
	Creator  string `json:"creator"`
	Status   int    `json:"status"`
	Response string `json:"response"`
	Page     int    `json:"page"`
}

type Item struct {
	Title     *string `json:"title"`
	Artwork   string  `json:"artwork"`
	Author    *string `json:"author"`
	AuthorURL *string `json:"authorUrl"`
	Page      int     `json:"page"`
}

type Payload struct {
	Data []Item `json:"data"`
}

// ==== API response ====

type apiItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Alt       string `json:"alt"`
	UserName  string `json:"userName"`
	UserID    string `json:"userId"`
	PageCount int    `json:"pageCount"`
}

type apiResp struct {
	Error bool `json:"error"`
	Body  struct {
		IllustManga struct {
			Data []apiItem `json:"data"`
		} `json:"illustManga"`
	} `json:"body"`
}

func ptrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ==== Search ====

func search(query string, page int) (Meta, Payload) {
	if page < 1 {
		page = 1
	}
	keyword := url.QueryEscape(query)

	u := fmt.Sprintf(
		"%s/ajax/search/artworks/%s?word=%s&order=date_d&mode=all&p=%d&csw=0&s_mode=s_tag_full&type=all&lang=en",
		baseURL, keyword, keyword, page,
	)

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return Meta{Creator: "Azén", Status: 500, Response: "Unknown", Page: page}, Payload{Data: []Item{}}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.pixiv.net/")

	res, err := searchClient.Do(req)
	if err != nil {
		return Meta{Creator: "Azén", Status: 500, Response: "Unknown", Page: page}, Payload{Data: []Item{}}
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	status := res.StatusCode

	var data apiResp
	_ = json.Unmarshal(body, &data)

	if status != 200 || data.Error {
		st := status
		if st == 200 {
			st = 500
		}
		respName := httpStatus[st]
		if respName == "" {
			respName = "Unknown"
		}
		return Meta{Creator: "Azén", Status: st, Response: respName, Page: page}, Payload{Data: []Item{}}
	}

	items := []Item{}
	for _, it := range data.Body.IllustManga.Data {
		title := it.Title
		if title == "" {
			title = it.Alt
		}
		authorURL := ""
		if it.UserID != "" {
			authorURL = baseURL + "/en/users/" + it.UserID
		}
		pc := it.PageCount
		if pc < 1 {
			pc = 1
		}
		items = append(items, Item{
			Title:     ptrIfNotEmpty(title),
			Artwork:   baseURL + "/en/artworks/" + it.ID,
			Author:    ptrIfNotEmpty(it.UserName),
			AuthorURL: ptrIfNotEmpty(authorURL),
			Page:      pc,
		})
	}

	return Meta{Creator: "Azén", Status: 200, Response: "OK", Page: page}, Payload{Data: items}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run pixiv_search.go <query> [page]")
		return
	}
	page := 1
	if len(os.Args) >= 3 {
		if p, err := strconv.Atoi(os.Args[2]); err == nil {
			page = p
		}
	}
	meta, payload := search(os.Args[1], page)
	out, _ := json.MarshalIndent([]any{meta, payload}, "", "  ")
	fmt.Println(string(out))
}
