# ripper

Rip audio from YouTube playlists.

## Requirements

- [ffmpeg](https://ffmpeg.org/download.html) on PATH
- `yt-dlp` is fetched automatically on first run

## Install

Grab a binary from [releases](https://github.com/Chitova263/YoutubePlaylistToAudioDownloaderCLI/releases) or build from source:

```bash
go build -o ripper .
```

## Usage

```bash
# single playlist
ripper rip --playlists "https://youtube.com/playlist?list=PLxxxxxxx"

# from a file (one URL per line)
ripper rip --input playlists.txt

# custom output dir, 4 concurrent downloads, wav format
ripper rip -i playlists.txt -o ~/Music --parallel 4 --audio-format wav
```

## Config

Pass `--config ./config.yml` or let it default to `$HOME/.ripper.yaml`.

```yaml
audio-format: mp3
output: ./downloads
parallel: 4
input: playlists.txt
```

CLI flags override config values.

## Flags

```
--audio-format string   output audio format (required)
-o, --output string     output directory (required)
-i, --input string      text file with playlist URLs (one per line)
--playlists string      comma separated playlist URLs
--parallel int          max parallel downloads (default 2)
--concurrency int       fragments downloaded concurrently per video (default 3)
--config string         path to config yaml (default $HOME/.ripper.yaml)
```

`--input` and `--playlists` are mutually exclusive; one is required.

`--parallel` controls how many videos download at once. `--concurrency` controls how many fragments each video fetches simultaneously. YouTube will rate-limit or block you if these are too aggressive, keep the defaults unless you know what you're doing.

Sweet spot: `--parallel 3 --concurrency 4`. Going above 5 concurrent fragments often causes failed downloads or HTTP 403s. More than 4-5 parallel videos and you're asking to get throttled.

## Output

```
<output>/
└── <Playlist Title>/
    ├── Track One.mp3
    └── Track Two.mp3
```

## License

© 2026 Nigel Mukandi
