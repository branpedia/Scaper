<div align="center">

# 🕷️ Scraper Tools

**Kumpulan API scraper berbasis Go** — siap dipakai untuk bot, website, dan aplikasi

> JSON output konsisten · Ringan · Zero config · Terus bertambah secara berkala ✨

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white)
![APIs](https://img.shields.io/badge/APIs-7+-blue?style=flat-square)
![Price](https://img.shields.io/badge/Price-Free-success?style=flat-square)
![API Key](https://img.shields.io/badge/API%20Key-Not%20Required-orange?style=flat-square)
![Creator](https://img.shields.io/badge/Creator-Bran-blueviolet?style=flat-square)

</div>

> 📦 **Build**: `go build -o scraper` — lalu jalankan `./scraper <cmd> <url>`

---

## 📑 Daftar Isi

- [🎯 API Tersedia](#-api-tersedia)
- [📦 Format Output](#-format-output)
- [🧪 Contoh Output per API](#-contoh-output-per-api)
  - [Instagram](#1️⃣-instagram-ig)
  - [TeraBox](#2️⃣-terabox-terabox)
  - [Pixiv Artwork](#3️⃣-pixiv-artwork-pixiv)
  - [Pixiv Search](#4️⃣-pixiv-search-pixivsearch)
  - [TikTok](#5️⃣-tiktok-tiktok)
  - [MediaFire](#6️⃣-mediafire-mediafire)
  - [Remove Background](#7️⃣-remove-background-removebg)
- [🤖 Integrasi Bot WA](#-integrasi-bot-wa-whatsapp-webjs)
- [🌐 Integrasi Web API](#-integrasi-web-api-express)
- [📋 Kebutuhan per API](#-kebutuhan-per-api)
- [🛠 Troubleshooting](#-troubleshooting)
- [⚠️ Etika & Legal](#️-etika--legal)
- [📜 Free & Open](#-free--open)

---

## 🎯 API Tersedia

| Command | Fungsi | Sumber |
|:---|:---|:---|
| `ig` | Instagram Reels / foto | indown.net |
| `terabox` | Resolve share TeraBox | sechno.com |
| `pixiv` | Artwork + download URL | pixiv.net |
| `pixivsearch` | Cari artwork by tag | pixiv.net ajax |
| `tiktok` | TikTok tanpa watermark | savetiktok.id |
| `mediafire` | Info file + direct link | mediafire.com |
| `removebg` | Hapus background gambar | iloveimg.com |

> 🚧 **Terus bertambah** — API baru akan ditambahkan secara berkala.
>
> 💡 **Hanya 2 dari 7** API yang butuh dependency eksternal (`goquery`): `ig` & `mediafire`. Sisanya **pure stdlib**.

---

## 📦 Format Output

Semua API mengembalikan **salah satu dari 2 format** konsisten:

### Format A — `status` / `result`

Dipakai oleh: `ig`, `terabox`, `pixiv`

**✅ Sukses**
```json
{
  "status": true,
  "result": { "...": "..." }
}
```

**❌ Gagal**
```json
{
  "status": false,
  "message": "pesan error"
}
```

### Format B — `[meta, payload]`

Dipakai oleh: `pixivsearch`, `tiktok`, `mediafire`, `removebg`

**✅ Sukses**
```json
[
  { "creator": "Bran", "status": 200, "response": "OK" },
  { "data": [ "...": "..." ] }
]
```

**❌ Gagal** — `status` mengikuti HTTP code (4xx / 5xx) + `data` kosong
```json
[
  { "creator": "Bran", "status": 400, "response": "Bad Request" },
  { "data": [] }
]
```

> ✅ Cek `status == 200` atau `status == true` untuk deteksi sukses. Gampang buat bot.

---

## 🧪 Contoh Output per API

### 1️⃣ Instagram (`ig`)

**Request**
```bash
./scraper ig "https://www.instagram.com/reel/CxyzAbCdEfG/"
```

**✅ Sukses**
```json
{
  "status": true,
  "result": {
    "title": "Instagram Media",
    "thumbnail": "https://scontent.cdninstagram.com/v/t51/thumb_reel.jpg",
    "downloads": [
      {
        "type": "video",
        "url": "https://snapcdn.workers.dev/dl?token=eyJhbG...mp4",
        "thumbnail": "https://scontent.cdninstagram.com/v/t51/thumb_reel.jpg"
      },
      {
        "type": "image",
        "url": "https://snapcdn.workers.dev/dl?token=eyJhbG...jpg",
        "thumbnail": "https://scontent.cdninstagram.com/v/t51/thumb_reel.jpg"
      }
    ]
  }
}
```

**❌ Gagal**
```json
{
  "status": false,
  "message": "No media links found. The post might be private or unavailable."
}
```

**💬 Kirim WA**
```
📸 *Instagram Downloader*
✅ Berhasil!

🎬 Video  : https://snapcdn...mp4
🖼️ Gambar : https://snapcdn...jpg
```

---

### 2️⃣ TeraBox (`terabox`)

**Request**
```bash
./scraper terabox "https://terabox.com/s/1AbCdEfGhIjK"
```

**✅ Sukses (multi-file)**
```json
{
  "status": true,
  "result": {
    "title": "Film Action 2024",
    "thumbnail": "https://data.terabox.com/thumbnail/film_action.jpg",
    "totalFiles": 3,
    "downloads": [
      {
        "type": "file",
        "filename": "Film.Action.2024.1080p.mp4",
        "size": "1.2 GB",
        "quality": "Direct Download",
        "url": "https://d.terabox.com/file/abc123"
      },
      {
        "type": "video",
        "filename": "Film.Action.2024.1080p.mp4",
        "size": "1.2 GB",
        "quality": "Fast Stream (m3u8)",
        "url": "https://d.terabox.com/stream/abc123.m3u8"
      },
      {
        "type": "file",
        "filename": "Subtitle.Indonesia.srt",
        "size": "45 KB",
        "quality": "Direct Download",
        "url": "https://d.terabox.com/file/def456"
      }
    ]
  }
}
```

**💬 Kirim WA**
```
📦 *TeraBox Downloader*
📁 Film Action 2024
📄 3 file ditemukan

1️⃣ Film.Action.2024.1080p.mp4 (1.2 GB)
   ⬇️ https://d.terabox.com/file/abc123
   ▶️ https://d.terabox.com/stream/abc123.m3u8

2️⃣ Subtitle.Indonesia.srt (45 KB)
   ⬇️ https://d.terabox.com/file/def456
```

---

### 3️⃣ Pixiv Artwork (`pixiv`)

**Request**
```bash
./scraper pixiv "https://www.pixiv.net/artworks/12345678"
```

**✅ Sukses (multi-page)**
```json
{
  "status": true,
  "result": {
    "title": "Sunset Girl by Azén",
    "thumbnail": "https://i.pixiv.re/c/540x540_70/img-master/img/.../12345678_p0_master1200.jpg",
    "type": "image",
    "author": {
      "name": "Azén",
      "id": "98765"
    },
    "downloads": [
      {
        "type": "image",
        "quality": "Page 1",
        "url": "https://i.pixiv.re/img-original/img/.../12345678_p0.jpg"
      },
      {
        "type": "image",
        "quality": "Page 2",
        "url": "https://i.pixiv.re/img-original/img/.../12345678_p1.jpg"
      }
    ]
  }
}
```

**✅ Sukses (ugoira / animasi)**
```json
{
  "status": true,
  "result": {
    "title": "Dancing Miku by Artist",
    "thumbnail": "https://pixiv.re/12345678.gif",
    "type": "video",
    "author": { "name": "Artist", "id": "11111" },
    "downloads": [
      {
        "type": "video",
        "quality": "MP4 Animation",
        "url": "https://ugoira.com/api/mp4/12345678"
      },
      {
        "type": "image",
        "quality": "GIF Animation",
        "url": "https://pixiv.re/12345678.gif"
      }
    ]
  }
}
```

---

### 4️⃣ Pixiv Search (`pixivsearch`)

**Request**
```bash
./scraper pixivsearch "hatsune miku" 1
```

**✅ Sukses**
```json
[
  {
    "creator": "Bran",
    "status": 200,
    "response": "OK",
    "page": 1
  },
  {
    "data": [
      {
        "title": "Miku Miku Dance",
        "artwork": "https://www.pixiv.net/en/artworks/12345678",
        "author": "Artist A",
        "authorUrl": "https://www.pixiv.net/en/users/11111",
        "page": 3
      },
      {
        "title": "Blue Miku",
        "artwork": "https://www.pixiv.net/en/artworks/87654321",
        "author": "Artist B",
        "authorUrl": "https://www.pixiv.net/en/users/22222",
        "page": 1
      }
    ]
  }
]
```

**✅ Sukses (hasil kosong)**
```json
[
  { "creator": "Bran", "status": 200, "response": "OK", "page": 1 },
  { "data": [] }
]
```

**⚠️ Rate limit**
```json
[
  { "creator": "Bran", "status": 429, "response": "Too Many Requests", "page": 1 },
  { "data": [] }
]
```

---

### 5️⃣ TikTok (`tiktok`)

**Request**
```bash
./scraper tiktok "https://www.tiktok.com/@user/video/7300000000000000000"
```

**✅ Sukses**
```json
[
  { "creator": "Bran", "status": 200, "response": "OK" },
  {
    "data": [
      {
        "title": "Dance challenge 🔥",
        "author": "@user",
        "duration": "00:15",
        "cover": "https://p16-sign-va.tiktokcdn.com/cover.jpg",
        "music_title": "Original Sound - user",
        "video_nowm": "https://v16-webapp.tiktokcdn.com/video_no_wm.mp4",
        "audio": "https://v16-webapp.tiktokcdn.com/audio.mp3",
        "filename_nowm": "tiktok_user_7300000000.mp4",
        "filename_audio": "tiktok_user_7300000000.mp3"
      }
    ]
  }
]
```

**❌ Gagal (private / invalid)**
```json
[
  { "creator": "Bran", "status": 400, "response": "Video not found or private" },
  { "data": [] }
]
```

**💬 Kirim WA**
```
🎵 *TikTok Downloader*
👤 @user
📝 Dance challenge 🔥
⏱️ 00:15

🎬 Video : https://v16...mp4
🎧 Audio : https://v16...mp3
```

---

### 6️⃣ MediaFire (`mediafire`)

**Request**
```bash
./scraper mediafire "https://www.mediafire.com/file/abc123/file.zip/file"
```

**✅ Sukses**
```json
[
  { "creator": "Bran", "status": 200, "response": "OK" },
  {
    "data": [
      {
        "filename": "file.zip",
        "filesize": "245.32MB",
        "uploaded": "2024-01-15 10:23:45",
        "filetype": "ZIP Archive",
        "download_url": "https://download1234.mediafire.com/xyzabc/file.zip"
      }
    ]
  }
]
```

**❌ Gagal**
```json
[
  { "creator": "Bran", "status": 400, "response": "Bad Request" },
  { "data": [] }
]
```

**💬 Kirim WA**
```
📁 *MediaFire Downloader*
📄 file.zip
💾 245.32MB · ZIP Archive
📅 2024-01-15

⬇️ https://download1234.mediafire.com/xyzabc/file.zip
```

---

### 7️⃣ Remove Background (`removebg`)

**Request**
```bash
./scraper removebg "./foto.jpg"
./scraper removebg "https://contoh.com/foto.jpg"
```

**✅ Sukses** — field `image` = **PNG base64**
```json
[
  { "creator": "Bran", "status": 200, "response": "OK" },
  {
    "data": [
      {
        "image": "iVBORw0KGgoAAAANSUhEUgAA...",
        "mime": "image/png"
      }
    ]
  }
]
```

> 💾 CLI `removebg` otomatis simpan ke `output_removebg.png`, jadi gak perlu decode manual.

**❌ Gagal**
```json
[
  { "creator": "Bran", "status": 500, "response": "Internal Server Error" },
  { "data": [] }
]
```

---

## 🤖 Integrasi Bot WA (whatsapp-web.js)

```js
const { Client } = require('whatsapp-web.js');
const { execFile } = require('child_process');

const client = new Client();

const handlers = {
  '!ig':        (u) => ['ig', u],
  '!terabox':   (u) => ['terabox', u],
  '!pixiv':     (u) => ['pixiv', u],
  '!tiktok':    (u) => ['tiktok', u],
  '!mediafire': (u) => ['mediafire', u],
  '!rmbg':      (u) => ['removebg', u],
};

function runScraper(cmd, url) {
  return new Promise((resolve, reject) => {
    execFile('./scraper', [cmd, url], { timeout: 60000 }, (err, stdout) => {
      if (err) return reject(err);
      try { resolve(JSON.parse(stdout)); }
      catch (e) { reject(e); }
    });
  });
}

client.on('message', async (msg) => {
  const [prefix, ...rest] = msg.body.trim().split(/\s+/);
  const handler = handlers[prefix];
  if (!handler) return;

  const [cmd, url] = handler(rest.join(' '));
  try {
    const res = await runScraper(cmd, url);
    await msg.reply('✅ ```' + JSON.stringify(res, null, 2) + '```');
  } catch (e) {
    await msg.reply('❌ ' + e.message);
  }
});

client.initialize();
```

---

## 🌐 Integrasi Web API (Express)

```js
const express = require('express');
const { execFile } = require('child_process');
const app = express();

const routes = ['ig', 'terabox', 'pixiv', 'tiktok', 'mediafire', 'removebg'];

routes.forEach((cmd) => {
  app.get(`/api/${cmd}`, (req, res) => {
    const url = req.query.url;
    if (!url) return res.status(400).json({ error: 'url required' });

    execFile('./scraper', [cmd, url], { timeout: 60000 }, (err, stdout) => {
      if (err) return res.status(500).json({ error: err.message });
      res.type('json').send(stdout);
    });
  });
});

app.get('/api/pixivsearch', (req, res) => {
  const { q, page = '1' } = req.query;
  if (!q) return res.status(400).json({ error: 'q required' });

  execFile('./scraper', ['pixivsearch', q, page], { timeout: 60000 }, (err, stdout) => {
    if (err) return res.status(500).json({ error: err.message });
    res.type('json').send(stdout);
  });
});

app.listen(3000);
```

**Contoh panggil:**

```bash
curl "http://localhost:3000/api/ig?url=https://instagram.com/reel/xxxx"
curl "http://localhost:3000/api/pixivsearch?q=hatsune+miku&page=1"
```

---

## 📋 Kebutuhan per API

| API | Input |
|:---|:---|
| `ig` | URL Instagram |
| `terabox` | URL TeraBox |
| `pixiv` | URL Pixiv |
| `pixivsearch` | tag + halaman |
| `tiktok` | URL TikTok |
| `mediafire` | URL MediaFire |
| `removebg` | URL gambar / file lokal |

> 💡 **Dependency**: hanya `ig` & `mediafire` yang butuh `github.com/PuerkitoBio/goquery`. Sisanya **pure stdlib** — zero dependency.

---

## 🛠 Troubleshooting

| Error | Solusi |
|:---|:---|
| `undefined: multipart.NewWriter` | Tambah `"mime/multipart"` di import `removebg.go` |
| `no required module provides goquery` | `go get github.com/PuerkitoBio/goquery` |
| `main redeclared` | 2 file `package main` di 1 folder → pindah folder |
| `429 Too Many Requests` | Rate limit — tunggu beberapa menit |
| `context deadline exceeded` | Server target lambat — ulangi |
| `Could not fetch Pixiv artwork data` | Artwork private / dihapus / butuh login |

---

## ⚠️ Etika & Legal

> 🚨 **Baca dulu sebelum pakai.**

### ✅ Boleh
- Gunakan untuk **konten publik** yang kamu berhak akses.
- Hormati **Terms of Service** tiap platform.
- Beri **delay ≥ 1 detik** antar request kalau batch.
- Pakai untuk **bot WA, bot Telegram, website, aplikasi pribadi**, dll.
- Modifikasi & pelajari kodenya  **bebas**.

### ❌ Jangan
- **Jangan menyerang / spam / DDOS** situs target. Scraper ini hanya untuk request normal, bukan alat serang.
- **Jangan redistribusi komersial tanpa izin.**
- **Jangan menghapus credit / mengganti creator lalu mengakui buatan sendiri.**

### 🔞 Catatan
- Konten **R-18 Pixiv** butuh login & session  scraper ini **tidak** menangani auth.

---

## 📜 Free & Open

Scraper ini **100% gratis** dan dibuat untuk keperluan:

- 🤖 Bot WhatsApp / Telegram / Discord
- 🌐 Backend website / REST API
- 🧪 Belajar & eksperimen pribadi
- 🛠️ Tools internal komunitas

**Dilarang diperjualbelikan**  kalau mau support, cukup **kasih credit** ke creator. 🙏

---

<div align="center">

**Made with ☕ by Bran**

⚠️ *Scraper ini mengambil data dari situs pihak ketiga.*
*Kalau situs sumbernya berubah, scraper mungkin perlu update.*

</div>
