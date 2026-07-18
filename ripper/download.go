package ripper

import (
	"YoutubePlaylistDownloader/logging"
	"bufio"
	"fmt"
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
	// ArchivePath is the path to a yt-dlp download archive file.
	// When set, yt-dlp records downloaded video IDs here and skips them on subsequent runs.
	ArchivePath string
}

// dlLog is the component logger for the download subsystem.
var dlLog = logging.Component("download")

func DownloadPlaylist(options DownloadOption) error {
	dlLog.Debug("invoking yt-dlp",
		"url", options.Url,
		"format", options.AudioFormat,
		"output_path", options.OutputFolderPath,
		"concurrent_fragments", options.ConcurrentFragments,
	)

	// -4 forces IPv4. Without it, yt-dlp may attempt IPv6 connections that hang
	// indefinitely on systems with broken or incomplete IPv6 connectivity.
	// Unlike curl (which uses "Happy Eyeballs" to race IPv6 and IPv4 simultaneously),
	// yt-dlp just tries one protocol and blocks. Forcing IPv4 sidesteps the issue.
	args := []string{
		"-4",
		"--extract-audio",
		"--audio-format", string(options.AudioFormat),
		"--audio-quality", options.AudioQuality,
		"--cookies", "./cookies.txt",
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
	}

	if options.ArchivePath != "" {
		args = append(args, "--download-archive", options.ArchivePath)
	}

	args = append(args, options.Url)

	cmd := exec.Command("./binaries/yt-dlp.exe", args...)

	logging.Trace("yt-dlp command", "args", cmd.Args)

	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("starting yt-dlp: %w", err)
	}

	dlLog.Debug("yt-dlp process started", "pid", cmd.Process.Pid, "url", options.Url)

	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		logging.Trace("yt-dlp stdout", "url", options.Url, "line", line)
		if strings.HasPrefix(line, "DONE") {
			dlLog.Debug("track file written", "url", options.Url, "output", line)
		}
	}

	err = cmd.Wait()
	if err != nil {
		dlLog.Error("yt-dlp exited with error", "url", options.Url, "error", err)
		return fmt.Errorf("yt-dlp error: %w", err)
	}

	dlLog.Debug("yt-dlp completed successfully", "url", options.Url)
	return nil
}
