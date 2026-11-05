package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

func printJSON(v any) {
	out, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(out))
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println(`usage:
  go run . ig          <url>
  go run . terabox     <url>
  go run . pixiv       <url>
  go run . pixivsearch <query> [page]
  go run . tiktok      <url>
  go run . mediafire   <url>
  go run . removebg    <url-or-path>`)
		return
	}

	cmd := os.Args[1]
	arg := os.Args[2]

	switch cmd {
	case "ig":
		printJSON(IG(arg))
	case "terabox":
		printJSON(TeraBox(arg))
	case "pixiv":
		printJSON(Pixiv(arg))
	case "pixivsearch":
		page := 1
		if len(os.Args) >= 4 {
			if p, err := strconv.Atoi(os.Args[3]); err == nil {
				page = p
			}
		}
		meta, payload := PixivSearch(arg, page)
		printJSON([]any{meta, payload})
	case "tiktok":
		meta, payload := TikTok(arg)
		printJSON([]any{meta, payload})
	case "mediafire":
		meta, payload := MediaFire(arg)
		printJSON([]any{meta, payload})
	case "removebg":
		meta, payload := RemoveBG(arg)
		printJSON([]any{meta, payload})
	default:
		fmt.Println("unknown command:", cmd)
	}
}
