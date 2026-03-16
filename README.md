# ytpd

CLI tool that downloads YouTube playlists as M4A audio files with rich metadata — cover art, composer credits, genre, label, and more.

## How it works

```
Excel (.xlsx)  →  YouTube scraping  →  yt-dlp download  →  MusicBrainz/Discogs lookup  →  ffmpeg metadata
```

1. Reads playlist URLs from an Excel file (column A)
2. Scrapes each YouTube playlist page for title and thumbnail
3. Downloads audio as M4A via `yt-dlp`
4. Looks up composer credits and metadata on **MusicBrainz** (with Discogs as fallback)
5. Embeds cover art + all metadata into M4A files via `ffmpeg`

## Usage

```bash
go build -o ytpd .
./ytpd <artist> <excel_file>
```

**Example:**

```bash
./ytpd "Kiko Loureiro" playlists.xlsx
```

The Excel file should have one playlist URL per row in column A:

| A |
|---|
| https://www.youtube.com/playlist?list=PLxxxxxx |
| https://www.youtube.com/playlist?list=PLyyyyyy |

## Output structure

```
Kiko Loureiro/
├── openSource/
│   ├── cover.jpg
│   ├── 01 - Gray Stone Gateway.m4a
│   ├── 02 - Backyard Trick.m4a
│   └── ...
└── sounds Of Innocence/
    ├── cover.jpg
    ├── 01 - The Anarchist.m4a
    └── ...
```

## Embedded metadata

Each M4A file gets tagged with all available metadata from MusicBrainz and Discogs:

| Tag | Description | ffmpeg key |
|-----|-------------|------------|
| Cover art | Playlist thumbnail | embedded as attached pic |
| Composer | Songwriter(s) | `composer` |
| Genre | Musical genre | `genre` |
| Date | Release year/date | `date` |
| Label | Record label | `----:com.apple.iTunes:LABEL` |
| Catalog # | Label catalog number | `----:com.apple.iTunes:CATALOGNUMBER` |
| Barcode | UPC/EAN | `----:com.apple.iTunes:BARCODE` |
| Country | Release country | `----:com.apple.iTunes:RELEASECOUNTRY` |
| Style | Sub-genres/tags | `----:com.apple.iTunes:STYLE` |
| Lyricist | Lyric writer(s) | `----:com.apple.iTunes:LYRICIST` |
| Arranger | Arranger(s) | `----:com.apple.iTunes:ARRANGER` |
| Producer | Producer(s) | `----:com.apple.iTunes:PRODUCER` |
| Engineer | Recording engineer(s) | `----:com.apple.iTunes:ENGINEER` |
| Mixer | Mixing engineer(s) | `----:com.apple.iTunes:MIXER` |
| ISRC | International Standard Recording Code | `----:com.apple.iTunes:ISRC` |
| MB Track ID | MusicBrainz recording ID | `----:com.apple.iTunes:MUSICBRAINZ_TRACKID` |
| MB Album ID | MusicBrainz release ID | `----:com.apple.iTunes:MUSICBRAINZ_ALBUMID` |
| MB Artist ID | MusicBrainz artist ID | `----:com.apple.iTunes:MUSICBRAINZ_ARTISTID` |

Verify embedded tags with:

```bash
ffprobe -show_entries format_tags -of json "01 - Gray Stone Gateway.m4a"
```

## Credit lookup strategy

1. **MusicBrainz** (primary) — free, no auth required
   - Searches recordings by artist + album + track title
   - Looks up work relations for composer/lyricist/arranger
   - Looks up artist relations for producer/engineer/mixer
   - Fetches release data for label, barcode, catalog number, genre
   - Rate limited to 1 request/second per their API policy

2. **Discogs** (fallback) — used when MusicBrainz finds no composers
   - Set `DISCOGS_TOKEN` env var for authenticated access
   - Searches releases by artist + album title
   - Extracts all credit roles from `extraartists` (release-level and track-level)
   - Also provides genre, styles, label, year, country, barcode

## Docker (sem instalar Go)

Rode o projeto sem precisar instalar Go, yt-dlp ou ffmpeg na sua máquina.

**Pré-requisitos:** Docker + Docker Compose.

```bash
# Build da imagem (só na primeira vez ou após mudanças no código)
docker compose build

# Rodar passando o artista e o caminho do Excel
ARTIST="Nome do Artista" EXCEL_PATH="./playlists.xlsx" docker compose run --rm app
```

**Exemplo:**

```bash
ARTIST="Kiko Loureiro" EXCEL_PATH="./playlists.xlsx" docker compose run --rm app
```

Os arquivos baixados aparecem em `./downloads/<artista>/` no seu PC.

> O container monta automaticamente os cookies do Firefox local para autenticação no YouTube.

---

## Prerequisites (execução local)

- **Go** 1.24+
- **yt-dlp** — `brew install yt-dlp`
- **ffmpeg** — `brew install ffmpeg`
- **YouTube cookies** — Firefox instalado localmente (usado via `--cookies-from-browser firefox`)

## Environment variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DISCOGS_TOKEN` | No | Discogs personal access token for fallback credit lookups. Get one at [discogs.com/settings/developers](https://www.discogs.com/settings/developers) |

## Development

```bash
# Run directly
go run main.go "Artist Name" playlists.xlsx

# Run tests
go test ./...

# Build
go build -o ytpd .
```

## Architecture

```
main.go                 CLI entry point
excel/reader.go         Reads playlist URLs from .xlsx
playlist/
  extractor.go          Scrapes YouTube playlist pages (goquery)
  processor.go          Orchestrates download + metadata pipeline
credits/
  credits.go            Types + fallback orchestration (MB → Discogs)
  musicbrainz.go        MusicBrainz API client
  discogs.go            Discogs API client
utils/
  file.go               SaveImage, FetchJSON, Sanitize, FormatDirName
  functional.go         Map, FilterMap generics
```
