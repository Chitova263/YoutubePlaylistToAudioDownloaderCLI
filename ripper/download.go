package ripper

import (
	"YoutubePlaylistDownloader/logging"
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
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
	// ArchivePath is the path to a yt-dlp download archive file.
	ArchivePath string
	// Ctx controls cancellation. When nil, context.Background() is used.
	Ctx context.Context
	// ProgressCallback is called with the download percentage (0–100) as
	// yt-dlp reports progress. May be nil.
	ProgressCallback func(percent float32)
}

// dlLog is the component logger for the download subsystem.
var dlLog = logging.Component("download")

func DownloadPlaylist(options DownloadOption) error {
	ctx := options.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	dlLog.Debug("invoking yt-dlp",
		"url", options.Url,
		"format", options.AudioFormat,
		"output_path", options.OutputFolderPath,
		"concurrent_fragments", options.ConcurrentFragments,
	)

	args := []string{
		"-4",
		"--extract-audio",
		"--audio-format", string(options.AudioFormat),
		"--audio-quality", options.AudioQuality,
	}

	if info, err := os.Stat("./cookies.txt"); err == nil && info.Size() > 0 {
		args = append(args, "--cookies", "./cookies.txt")
	}

	args = append(args,
		"-f", options.FormatSelector,
		"--ffmpeg-location", "./binaries",
		"--embed-thumbnail",
		"--embed-metadata",
		"--concurrent-fragments", strconv.Itoa(int(options.ConcurrentFragments)),
		"--retries", "3",
		"--fragment-retries", "5",
		"--parse-metadata", "title:(?P<meta_artist>.+) - (?P<meta_title>.+)",
		"--parse-metadata", "%(artist,creator,meta_artist)s:%(meta_artist)s",
		"--print", "after_move:DONE %(id)s %(filepath)q",
		"-o", options.OutputFolderPath,
	)

	if options.ArchivePath != "" {
		args = append(args, "--download-archive", options.ArchivePath)
	}

	args = append(args, options.Url)

	cmd := exec.CommandContext(ctx, "./binaries/yt-dlp.exe", args...)

	logging.Trace("yt-dlp command", "args", cmd.Args)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}

	if err = cmd.Start(); err != nil {
		return fmt.Errorf("starting yt-dlp: %w", err)
	}

	dlLog.Debug("yt-dlp process started", "pid", cmd.Process.Pid, "url", options.Url)

	var pipeWg sync.WaitGroup
	pipeWg.Add(2)

	// Drain stdout — watch for DONE signals from --print.
	go func() {
		defer pipeWg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			logging.Trace("yt-dlp stdout", "url", options.Url, "line", line)
			if strings.HasPrefix(line, "DONE") {
				dlLog.Debug("track file written", "url", options.Url, "output", line)
			}
		}
	}()

	// Drain stderr — parse [download] progress lines.
	go func() {
		defer pipeWg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			logging.Trace("yt-dlp stderr", "url", options.Url, "line", line)
			if options.ProgressCallback != nil {
				if pct, ok := parseDownloadPercent(line); ok {
					options.ProgressCallback(pct)
				}
			}
		}
	}()

	// Wait for both pipes to be fully drained before calling Wait.
	pipeWg.Wait()

	if err = cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		dlLog.Error("yt-dlp exited with error", "url", options.Url, "error", err)
		return fmt.Errorf("yt-dlp error: %w", err)
	}

	dlLog.Debug("yt-dlp completed successfully", "url", options.Url)
	return nil
}

// parseDownloadPercent extracts the percentage from a yt-dlp progress line.
// Example: "[download]  45.2% of   3.50MiB at  1.20MiB/s ETA 00:02"
// Returns (percent, true) on success, (0, false) if the line is not a progress line.
func parseDownloadPercent(line string) (float32, bool) {
	if !strings.Contains(line, "[download]") {
		return 0, false
	}
	pctIdx := strings.Index(line, "%")
	if pctIdx < 0 {
		return 0, false
	}
	before := strings.TrimSpace(line[:pctIdx])
	fields := strings.Fields(before)
	if len(fields) == 0 {
		return 0, false
	}
	numStr := fields[len(fields)-1]
	val, err := strconv.ParseFloat(numStr, 32)
	if err != nil {
		return 0, false
	}
	// Cap at 99.9 — we emit 100 only via TrackComplete so the UI stays consistent.
	if val > 99.9 {
		val = 99.9
	}
	return float32(val), true
}
