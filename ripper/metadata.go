package ripper

import (
	"YoutubePlaylistDownloader/logging"
	"YoutubePlaylistDownloader/playlist"
	"encoding/json"
	"fmt"
	"os/exec"
)

// metaLog is the component logger for metadata extraction.
var metaLog = logging.Component("metadata")

type PlaylistMetadataDownloadOptions struct {
	PlayListUrl string
}

func DownloadPlaylistMetadata(opts PlaylistMetadataDownloadOptions) (playlist.PlaylistMetadata, error) {
	metaLog.Debug("fetching playlist metadata", "url", opts.PlayListUrl)

	// -4 forces IPv4 (see download.go for rationale)
	cmd := exec.Command(
		"./binaries/yt-dlp.exe",
		"-4",
		"--flat-playlist", "-J",
		"--no-warnings",
		"--ignore-errors",
		"--yes-playlist",
		"--skip-playlist-after-errors", "5",
		opts.PlayListUrl,
	)

	logging.Trace("yt-dlp metadata command", "args", cmd.Args)

	output, err := cmd.Output()
	if err != nil {
		return playlist.PlaylistMetadata{}, fmt.Errorf("yt-dlp metadata fetch failed: %w", err)
	}

	logging.Trace("raw metadata response", "bytes", len(output))

	var metadata playlist.PlaylistMetadata
	if err := json.Unmarshal(output, &metadata); err != nil {
		return playlist.PlaylistMetadata{}, fmt.Errorf("parsing playlist metadata: %w", err)
	}

	metaLog.Debug("metadata parsed",
		"playlist_id", metadata.ID,
		"title", metadata.Title,
		"entries", len(metadata.Entries),
	)

	return metadata, nil
}
