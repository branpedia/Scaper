package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"       
	"net"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	rbPageURL = "https://www.iloveimg.com/remove-background"
	rbAPIBase = "https://api6g.iloveimg.com/v1"
	rbUA      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)

var rbStatus = map[int]string{
	200: "OK", 301: "Moved Permanently", 302: "Found",
	400: "Bad Request", 403: "Forbidden", 404: "Not Found",
	429: "Too Many Requests", 500: "Internal Server Error", 503: "Service Unavailable",
}

var rbClient = &http.Client{
	Timeout: 120 * time.Second,
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
			d := net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
			return d.DialContext(ctx, "tcp4", addr)
		},
		ForceAttemptHTTP2: true,
	},
}

var (
	rbReTask  = regexp.MustCompile(`ilovepdfConfig\.taskId\s*=\s*'([^']+)'`)
	rbReToken = regexp.MustCompile(`token["']?\s*[:=]\s*["']([A-Za-z0-9._-]{20,})["']`)
)

type RBMeta struct {
	Creator  string `json:"creator"`
	Status   int    `json:"status"`
	Response string `json:"response"`
}

type RBResult struct {
	Image string `json:"image"`
	Mime  string `json:"mime"`
}

type RBPayload struct {
	Data []RBResult `json:"data"`
}

func rbStatusName(code int) string {
	if s, ok := rbStatus[code]; ok {
		return s
	}
	return "Unknown"
}

func rbFail(status int, resp string) (RBMeta, RBPayload) {
	return RBMeta{"Bran", status, resp}, RBPayload{Data: []RBResult{}}
}

func rbResolve(input string) ([]byte, string, string, error) {
	if strings.HasPrefix(input, "http") {
		req, _ := http.NewRequest("GET", input, nil)
		req.Header.Set("User-Agent", rbUA)
		res, err := rbClient.Do(req)
		if err != nil {
			return nil, "", "", err
		}
		defer res.Body.Close()
		if res.StatusCode >= 300 {
			return nil, "", "", fmt.Errorf("download failed: %d", res.StatusCode)
		}
		buf, _ := io.ReadAll(res.Body)
		mimeType := strings.Split(res.Header.Get("Content-Type"), ";")[0]
		if mimeType == "" {
			mimeType = "image/jpeg"
		}
		ext := "jpg"
		switch {
		case strings.Contains(mimeType, "png"):
			ext = "png"
		case strings.Contains(mimeType, "webp"):
			ext = "webp"
		}
		return buf, "image." + ext, mimeType, nil
	}
	if _, err := os.Stat(input); err != nil {
		return nil, "", "", fmt.Errorf("File tidak ditemukan: %s", input)
	}
	buf, _ := os.ReadFile(input)
	fname := filepath.Base(input)
	mimeType := "image/jpeg"
	if strings.HasSuffix(strings.ToLower(fname), ".png") {
		mimeType = "image/png"
	}
	return buf, fname, mimeType, nil
}

func rbSession() (string, string, error) {
	req, _ := http.NewRequest("GET", rbPageURL, nil)
	req.Header.Set("User-Agent", rbUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	res, err := rbClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	html := string(body)
	tm := rbReTask.FindStringSubmatch(html)
	if tm == nil {
		return "", "", fmt.Errorf("Gagal ambil taskId dari halaman iLoveIMG")
	}
	tok := rbReToken.FindStringSubmatch(html)
	if tok == nil {
		return "", "", fmt.Errorf("Gagal ambil token dari halaman iLoveIMG")
	}
	return tm[1], tok[1], nil
}

func removeBackground(input string) (RBMeta, RBPayload) {
	if input == "" {
		return rbFail(400, "Bad Request")
	}

	buffer, fname, mimeType, err := rbResolve(input)
	if err != nil {
		return rbFail(500, "Internal Server Error")
	}
	taskID, token, err := rbSession()
	if err != nil {
		return rbFail(500, "Internal Server Error")
	}

	// upload
	var upBuf bytes.Buffer
	w := multipart.NewWriter(&upBuf)
	w.WriteField("task", taskID)
	w.WriteField("preview", "0")
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, fname))
	h.Set("Content-Type", mimeType)
	part, _ := w.CreatePart(h)
	part.Write(buffer)
	w.Close()

	upReq, _ := http.NewRequest("POST", rbAPIBase+"/upload", &upBuf)
	upReq.Header.Set("Content-Type", w.FormDataContentType())
	upReq.Header.Set("Authorization", "Bearer "+token)
	upReq.Header.Set("User-Agent", rbUA)
	upReq.Header.Set("Origin", "https://www.iloveimg.com")
	upReq.Header.Set("Referer", "https://www.iloveimg.com/")

	upRes, err := rbClient.Do(upReq)
	if err != nil {
		return rbFail(500, "Internal Server Error")
	}
	defer upRes.Body.Close()
	if upRes.StatusCode >= 300 {
		return rbFail(upRes.StatusCode, rbStatusName(upRes.StatusCode))
	}

	upBody, _ := io.ReadAll(upRes.Body)
	var up struct {
		ServerFilename string `json:"server_filename"`
	}
	_ = json.Unmarshal(upBody, &up)
	if up.ServerFilename == "" {
		return rbFail(500, "Internal Server Error")
	}

	// remove bg
	var taskBuf bytes.Buffer
	tw := multipart.NewWriter(&taskBuf)
	tw.WriteField("task", taskID)
	tw.WriteField("server_filename", up.ServerFilename)
	tw.Close()

	taskReq, _ := http.NewRequest("POST", rbAPIBase+"/removebackground", &taskBuf)
	taskReq.Header.Set("Content-Type", tw.FormDataContentType())
	taskReq.Header.Set("Authorization", "Bearer "+token)
	taskReq.Header.Set("User-Agent", rbUA)
	taskReq.Header.Set("Origin", "https://www.iloveimg.com")
	taskReq.Header.Set("Referer", "https://www.iloveimg.com/")

	taskRes, err := rbClient.Do(taskReq)
	if err != nil {
		return rbFail(500, "Internal Server Error")
	}
	defer taskRes.Body.Close()
	if taskRes.StatusCode >= 300 {
		return rbFail(taskRes.StatusCode, rbStatusName(taskRes.StatusCode))
	}

	imgBuf, _ := io.ReadAll(taskRes.Body)
	mimeOut := strings.Split(taskRes.Header.Get("Content-Type"), ";")[0]
	if mimeOut == "" {
		mimeOut = "image/png"
	}

	result := RBResult{
		Image: base64.StdEncoding.EncodeToString(imgBuf),
		Mime:  mimeOut,
	}
	return RBMeta{"Bran", 200, "OK"}, RBPayload{Data: []RBResult{result}}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run removebg.go <url-or-filepath>")
		return
	}
	meta, payload := removeBackground(os.Args[1])

	if meta.Status == 200 && len(payload.Data) > 0 {
		raw, _ := base64.StdEncoding.DecodeString(payload.Data[0].Image)
		_ = os.WriteFile("output_removebg.png", raw, 0644)
		fmt.Println("✓ Disimpan ke output_removebg.png")
	}

	out, _ := json.MarshalIndent([]any{meta, payload}, "", "  ")
	fmt.Println(string(out))
}
