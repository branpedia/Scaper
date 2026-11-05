package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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
	ilovePageURL = "https://www.iloveimg.com/remove-background"
	iloveAPIBase = "https://api6g.iloveimg.com/v1"
	iloveUA      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)

var iloveHTTPStatus = map[int]string{
	200: "OK", 301: "Moved Permanently", 302: "Found",
	400: "Bad Request", 403: "Forbidden", 404: "Not Found",
	429: "Too Many Requests", 500: "Internal Server Error", 503: "Service Unavailable",
}

// Client dengan IPv4 forced (setara HttpsAgent family:4 di JS)
var iloveClient = &http.Client{
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
	reTaskID = regexp.MustCompile(`ilovepdfConfig\.taskId\s*=\s*'([^']+)'`)
	reToken  = regexp.MustCompile(`token["']?\s*[:=]\s*["']([A-Za-z0-9._-]{20,})["']`)
)

// ==== Output ====

type iloveMeta struct {
	Creator  string `json:"creator"`
	Status   int    `json:"status"`
	Response string `json:"response"`
}

type iloveResult struct {
	Image string `json:"image"` // base64-encoded PNG
	Mime  string `json:"mime"`
}

type ilovePayload struct {
	Data []iloveResult `json:"data"`
}

func iloveStatusName(code int) string {
	if s, ok := iloveHTTPStatus[code]; ok {
		return s
	}
	return "Unknown"
}

func iloveFail(status int, resp string) (iloveMeta, ilovePayload) {
	return iloveMeta{Creator: "Bran", Status: status, Response: resp},
		ilovePayload{Data: []iloveResult{}}
}

// ==== Resolve input (URL atau file path) ====

func resolveImage(input string) ([]byte, string, string, error) {
	// URL
	if strings.HasPrefix(input, "http") {
		req, _ := http.NewRequest("GET", input, nil)
		req.Header.Set("User-Agent", iloveUA)
		res, err := iloveClient.Do(req)
		if err != nil {
			return nil, "", "", err
		}
		defer res.Body.Close()
		if res.StatusCode >= 300 {
			return nil, "", "", fmt.Errorf("download failed: %d", res.StatusCode)
		}
		buf, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, "", "", err
		}
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

	// File path
	if _, err := os.Stat(input); err != nil {
		return nil, "", "", fmt.Errorf("File tidak ditemukan: %s", input)
	}
	buf, err := os.ReadFile(input)
	if err != nil {
		return nil, "", "", err
	}
	filename := filepath.Base(input)
	mimeType := "image/jpeg"
	if strings.HasSuffix(strings.ToLower(filename), ".png") {
		mimeType = "image/png"
	}
	return buf, filename, mimeType, nil
}

// ==== Ambil taskId + token dari halaman tool ====

func iloveGetSession() (string, string, error) {
	req, _ := http.NewRequest("GET", ilovePageURL, nil)
	req.Header.Set("User-Agent", iloveUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	res, err := iloveClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	html := string(body)

	tm := reTaskID.FindStringSubmatch(html)
	if tm == nil {
		return "", "", fmt.Errorf("Gagal ambil taskId dari halaman iLoveIMG")
	}
	tok := reToken.FindStringSubmatch(html)
	if tok == nil {
		return "", "", fmt.Errorf("Gagal ambil token dari halaman iLoveIMG")
	}
	return tm[1], tok[1], nil
}

// ==== Multipart builder helper ====

func buildUploadBody(taskID string, buffer []byte, filename, mimetype string) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("task", taskID)
	_ = w.WriteField("preview", "0")

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	h.Set("Content-Type", mimetype)
	part, err := w.CreatePart(h)
	if err != nil {
		return nil, "", err
	}
	part.Write(buffer)
	w.Close()
	return &buf, w.FormDataContentType(), nil
}

func buildTaskBody(taskID, serverFilename string) (*bytes.Buffer, string) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("task", taskID)
	w.WriteField("server_filename", serverFilename)
	w.Close()
	return &buf, w.FormDataContentType()
}

// ==== Main ====

func removeBackground(input string) (iloveMeta, ilovePayload) {
	if input == "" {
		return iloveFail(400, "Bad Request")
	}

	// 1. Resolve input
	buffer, filename, mimetype, err := resolveImage(input)
	if err != nil {
		return iloveFail(500, "Internal Server Error")
	}

	// 2. Ambil session
	taskID, token, err := iloveGetSession()
	if err != nil {
		return iloveFail(500, "Internal Server Error")
	}

	// 3. Upload
	upBody, upCT, err := buildUploadBody(taskID, buffer, filename, mimetype)
	if err != nil {
		return iloveFail(500, "Internal Server Error")
	}
	upReq, _ := http.NewRequest("POST", iloveAPIBase+"/upload", upBody)
	upReq.Header.Set("Content-Type", upCT)
	upReq.Header.Set("Authorization", "Bearer "+token)
	upReq.Header.Set("User-Agent", iloveUA)
	upReq.Header.Set("Origin", "https://www.iloveimg.com")
	upReq.Header.Set("Referer", "https://www.iloveimg.com/")

	upRes, err := iloveClient.Do(upReq)
	if err != nil {
		return iloveFail(500, "Internal Server Error")
	}
	defer upRes.Body.Close()

	if upRes.StatusCode >= 300 {
		return iloveFail(upRes.StatusCode, iloveStatusName(upRes.StatusCode))
	}

	upBytes, _ := io.ReadAll(upRes.Body)
	var up struct {
		ServerFilename string `json:"server_filename"`
	}
	_ = json.Unmarshal(upBytes, &up)
	if up.ServerFilename == "" {
		return iloveFail(500, "Internal Server Error")
	}

	// 4. Remove background
	taskBody, taskCT := buildTaskBody(taskID, up.ServerFilename)
	taskReq, _ := http.NewRequest("POST", iloveAPIBase+"/removebackground", taskBody)
	taskReq.Header.Set("Content-Type", taskCT)
	taskReq.Header.Set("Authorization", "Bearer "+token)
	taskReq.Header.Set("User-Agent", iloveUA)
	taskReq.Header.Set("Origin", "https://www.iloveimg.com")
	taskReq.Header.Set("Referer", "https://www.iloveimg.com/")

	taskRes, err := iloveClient.Do(taskReq)
	if err != nil {
		return iloveFail(500, "Internal Server Error")
	}
	defer taskRes.Body.Close()

	if taskRes.StatusCode >= 300 {
		return iloveFail(taskRes.StatusCode, iloveStatusName(taskRes.StatusCode))
	}

	imgBuf, _ := io.ReadAll(taskRes.Body)
	mimeType := strings.Split(taskRes.Header.Get("Content-Type"), ";")[0]
	if mimeType == "" {
		mimeType = "image/png"
	}

	result := iloveResult{
		Image: base64.StdEncoding.EncodeToString(imgBuf),
		Mime:  mimeType,
	}
	return iloveMeta{Creator: "Bran", Status: 200, Response: "OK"},
		ilovePayload{Data: []iloveResult{result}}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run removebg.go <url-or-filepath>")
		return
	}
	meta, payload := removeBackground(os.Args[1])

	// Simpan PNG hasil ke file biar gak kepanjangan di terminal
	if meta.Status == 200 && len(payload.Data) > 0 {
		raw, _ := base64.StdEncoding.DecodeString(payload.Data[0].Image)
		_ = os.WriteFile("output_removebg.png", raw, 0644)
		fmt.Println("✓ Disimpan ke output_removebg.png")
	}

	// Untuk meta/payload di JSON, ganti image ke placeholder biar ringkas
	out, _ := json.MarshalIndent([]any{meta, ilovePayload{Data: payload.Data}}, "", "  ")
	fmt.Println(string(out))
}
