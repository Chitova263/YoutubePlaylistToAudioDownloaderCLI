package ytdlp

import (
	"YoutubePlaylistDownloader/playlist"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
)

type PlaylistMetadataDownloadOptions struct {
	PlayListUrl string
}

func DownloadPlaylistMetadata(playlistMetadataDownloadOptions PlaylistMetadataDownloadOptions) (playlist.PlaylistMetadata, error) {

	slog.Debug("Downloading Playlist metadata", "options", playlistMetadataDownloadOptions)

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
		"--flat-playlist", "-J",
		"--no-warnings",
		"--ignore-errors",
		"--yes-playlist",
		"--skip-playlist-after-errors", "5",
		playlistMetadataDownloadOptions.PlayListUrl,
	)

	output, err := cmd.Output()
	if err != nil {
		return playlist.PlaylistMetadata{}, fmt.Errorf("error downloading playlist metadata: %w\noutput: %s", err, output)
	}
	var playlistMetadata playlist.PlaylistMetadata
	err = json.Unmarshal(output, &playlistMetadata)
	if err != nil {
		return playlist.PlaylistMetadata{}, fmt.Errorf("error parsing playlist metadata: %w\noutput: %s", err, output)
	}

	slog.Debug("Downloaded playlist metadata", "metadata", playlistMetadata)

	return playlistMetadata, nil
}
