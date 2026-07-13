# ytpdl

A CLI tool to download YouTube playlists and convert them to audio files (MP3, WAV, etc.) for offline listening.

## Features

- Download entire YouTube playlists as audio files
- Concurrent downloads with configurable parallelism
- Supports multiple audio formats (MP3, WAV, etc.)
- Batch processing via playlist files (one URL per line)
- Automatic yt-dlp binary management (downloads if missing)
- Cross-platform support (Windows, Linux, macOS)

## Prerequisites

- [ffmpeg](https://ffmpeg.org/download.html) installed and available on PATH

> **Note:** `yt-dlp` is automatically downloaded on first run if not already present in the `binaries/` directory.

## Installation

```bash
go install github.com/your-username/YoutubePlaylistDownloader@latest
```

Or build from source:

```bash
git clone https://github.com/your-username/YoutubePlaylistDownloader.git
cd YoutubePlaylistDownloader
go build -o ytpdl .
```

## Usage

### Download a playlist

```bash
ytpdl rip --playlist playlists.txt
```

Where `playlists.txt` contains one YouTube playlist URL per line:

```
https://youtube.com/playlist?list=PLxxxxxxx
https://youtube.com/playlist?list=PLyyyyyyy
```

### Options

```
Flags:
      --url string                    YouTube playlist url
      --format string                 Output audio format (default "mp3")
      --output string                 Output directory (default ".")
      --playlist string               Path to a text file containing playlist URLs
      --max-parallel-downloads int    Maximum number of parallel downloads (default 2)
  -h, --help                          Help for rip
```

### Examples

```bash
# Download playlists listed in a file as MP3
ytpdl rip --playlist playlists.txt

# Download to a specific directory with 4 concurrent downloads
ytpdl rip --playlist playlists.txt --output ~/Music --max-parallel-downloads 4

# Download a single playlist via URL
ytpdl rip --url "https://youtube.com/playlist?list=PLxxxxxxx"

# Download as WAV format
ytpdl rip --url "https://youtube.com/playlist?list=PLxxxxxxx" --format wav
```

## Output Structure

Downloaded files are saved using the output template `<output>/<Playlist Title>/<Track Title>.<ext>`:

```
<output-dir>/
└── <Playlist Title>/
    ├── Track One.mp3
    ├── Track Two.mp3
    └── Track Three.mp3
```

## How It Works

1. **Metadata extraction** — Fetches playlist metadata (titles, video IDs, URLs) using yt-dlp
2. **Concurrent download** — Downloads each track in parallel (limited by `--max-parallel-downloads`)
3. **Audio conversion** — Converts video streams to the specified audio format via ffmpeg

## Tech Stack

- **Language:** Go 1.25
- **CLI framework:** [Cobra](https://github.com/spf13/cobra)
- **Download engine:** [yt-dlp](https://github.com/yt-dlp/yt-dlp)
- **Audio processing:** [ffmpeg](https://ffmpeg.org/)

## License

Copyright © 2026 Nigel Mukandi
