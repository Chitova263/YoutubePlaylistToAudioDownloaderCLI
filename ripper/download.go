package ripper

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type AudioFormat string

const (
	FormatBest AudioFormat = "best"
	FormatMP3  AudioFormat = "mp3"
	FormatM4A  AudioFormat = "m4a"
	FormatOpus AudioFormat = "opus"
	FormatFLAC AudioFormat = "flac"
)

type DownloadOption struct {
	AudioFormat         AudioFormat
	OutputFolderPath    string
	ConcurrentFragments uint
	Thumbnail           bool
	Url                 string
	FormatSelector      string
	AudioQuality        string
}

func DownloadPlaylist(options DownloadOption) error {
	slog.Info("executing yt-dlp", "url", options.Url, "format", options.AudioFormat, "output_path", options.OutputFolderPath)
	// -4 forces IPv4. Without it, yt-dlp may attempt IPv6 connections that hang
	// indefinitely on systems with broken or incomplete IPv6 connectivity.
	//
	// Many machines have IPv6 enabled at the OS level but lack a fully functional
	// end-to-end IPv6 route to YouTube's servers. When yt-dlp resolves youtube.com,
	// the OS returns both IPv6 and IPv4 addresses. yt-dlp tries IPv6 first (preferred
	// by default), but if the connection attempt goes into a black hole (no response,
	// no rejection), it waits forever since there's no built-in timeout.
	//
	// Unlike curl (which uses "Happy Eyeballs" to race IPv6 and IPv4 simultaneously),
	// yt-dlp just tries one protocol and blocks. Forcing IPv4 sidesteps the issue entirely.
	cmd := exec.Command(
		"./binaries/yt-dlp.exe",
		"-4",
		"--extract-audio",
		//  strips video, keeps only audio. Requires ffmpeg/ffprobe installed.
		"--audio-format", string(options.AudioFormat),
		// 0 (best, VBR) to 10 (worst), or an explicit bitrate like 192K. Use 0 for maximum, since you said the highest quality. If you want deterministic file sizes across a big library, use a fixed constant bitrate like 320K instead of VBR 0
		"--audio-quality", options.AudioQuality,
		"--cookies", "./cookies.txt",
		// format selector. bestaudio picks the highest-bitrate audio-only stream (usually 160-256kbps Opus); falls back to best (a muxed video+audio stream) only if no audio-only stream exists. This is the right selector for "highest quality audio" — don't hardcode a format id, YouTube's available streams change per video.
		"-f", options.FormatSelector,
		// embeds the video thumbnail as cover art (needs mutagen or ffmpeg). This is what gives you the "Spotify/iTunes-style" look with album art in Serato/Rekordbox browsers.
		"--embed-thumbnail",
		"--embed-metadata",
		// how many pieces of a single video's stream download in parallel (default 1). Keep this modest — 4 is a reasonable ceiling. Going much higher (people report issues around 16) from one IP is what tends to trigger throttling/blocks.
		"--concurrent-fragments", strconv.Itoa(int(options.ConcurrentFragments)),
		"--parse-metadata", "title:(?P<meta_artist>.+) - (?P<meta_title>.+)",
		"--parse-metadata", "%(artist,creator,meta_artist)s:%(meta_artist)s",
		// after_move is the only one guaranteed to fire after metadata embedding, thumbnail embedding, and the final file rename/move to its output path are all complete
		"--print", "after_move:DONE %(id)s %(filepath)q",
		// Output template: saves files into <OutputFolderPath>/<playlist name>/<title>.<ext>.
		// %(playlist)s resolves to the playlist title. If the URL is a single video (no playlist),
		// it falls back to the --output-na-placeholder value ("NA" by default).
		"-o", options.OutputFolderPath,
		options.Url,
	)

	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %w", err)
	}

	// cmd.Run() -> cmd.Start() + cmd.Wait()
	// We dont want blocking until the process fully exits so we use cmd.Start()
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("error downloading playlist: %w", err)
	}

	// Read line by line from the pipe to capture yt-dlp output on std out
	scanner := bufio.NewScanner(stdout)
	// blocks until the underlying reader hits EOF
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		slog.Debug("yt-dlp output", "url", options.Url, "line", line)
		// Single file completed downloading if line starts with DONE
		if strings.HasPrefix(line, "DONE") {
			slog.Info("track download complete", "url", options.Url, "output", line)
		}
	}

	// Block until process exits
	err = cmd.Wait()
	if err != nil {
		slog.Error("yt-dlp process exited with error", "url", options.Url, "error", err)
		return fmt.Errorf("download error: %w", err)
	}
	return nil
}
