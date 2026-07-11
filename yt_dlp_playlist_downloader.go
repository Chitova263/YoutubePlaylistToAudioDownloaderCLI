package main

import (
	"bufio"
	"fmt"
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

type PlaylistDownloadOption struct {
	AudioFormat         AudioFormat
	OutputFolderPath    string
	ConcurrentFragments uint
	Thumbnail           bool
	PlayListUrl         string
	FormatSelector      string
	AudioQuality        string
}

func DownloadPlaylist(options PlaylistDownloadOption) error {

	cmd := exec.Command(
		"./binaries/yt-dlp.exe",
		"--extract-audio",
		//  strips video, keeps only audio. Requires ffmpeg/ffprobe installed.
		"--audio-format", string(options.AudioFormat),
		// 0 (best, VBR) to 10 (worst), or an explicit bitrate like 192K. Use 0 for maximum, since you said the highest quality. If you want deterministic file sizes across a big library, use a fixed constant bitrate like 320K instead of VBR 0
		"--audio-quality", options.AudioQuality,
		// format selector. bestaudio picks the highest-bitrate audio-only stream (usually 160-256kbps Opus); falls back to best (a muxed video+audio stream) only if no audio-only stream exists. This is the right selector for "highest quality audio" — don't hardcode a format id, YouTube's available streams change per video.
		"-f", options.FormatSelector,
		// embeds the video thumbnail as cover art (needs mutagen or ffmpeg). This is what gives you the "Spotify/iTunes-style" look with album art in Serato/Rekordbox browsers.
		"--embed-thumbnail",
		"--embed-metadata",
		// how many pieces of a single video's stream download in parallel (default 1). Keep this modest — 4 is a reasonable ceiling. Going much higher (people report issues around 16) from one IP is what tends to trigger throttling/blocks.
		"--concurrent-fragments", strconv.Itoa(int(options.ConcurrentFragments)),
		"--parse-metadata", "%(artist,creator)s:%(meta_artist)s",
		// after_move is the only one guaranteed to fire after metadata embedding, thumbnail embedding, and the final file rename/move to its output path are all complete
		"--print", "after_move:DONE %(id)s %(filepath)q",
		// Output template: saves files into <OutputFolderPath>/<playlist name>/<title>.<ext>.
		// %(playlist)s resolves to the playlist title. If the URL is a single video (no playlist),
		// it falls back to the --output-na-placeholder value ("NA" by default).
		"-o", getOutputTemplate(options.OutputFolderPath),
		options.PlayListUrl,
	)

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
		fmt.Println(line)
		// Single file completed downloading if line starts with DONE
		if strings.HasPrefix(line, "DONE") {
			// TODO pass trackId to the next stage in pipeline
			fmt.Println("DONE........ NEXT PIPELINE")
		}
	}

	// Block until process exits
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("error downloading playlist: %w", err)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error downloading playlist: %w\noutput: %s", err, output)
	}
	return nil
}

func getOutputTemplate(path string) string {
	return path + "/%(playlist)s/%(title)s.%(ext)s"
}
