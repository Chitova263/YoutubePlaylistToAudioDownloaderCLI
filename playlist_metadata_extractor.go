package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type PlaylistMetadataDownloadOptions struct {
	PlayListUrl             string
	SkipPlayListAfterErrors string
}

func DownloadPlaylistMetadata(playlistMetadataDownloadOptions PlaylistMetadataDownloadOptions) (PlaylistMetadata, error) {
	cmd := exec.Command(
		"./binaries/yt-dlp.exe",
		"--flat-playlist", "-J",
		"--no-warnings",
		"--ignore-errors",
		"--skip-playlist-after-errors", playlistMetadataDownloadOptions.SkipPlayListAfterErrors,
		playlistMetadataDownloadOptions.PlayListUrl,
	)

	output, err := cmd.Output()
	if err != nil {
		return PlaylistMetadata{}, fmt.Errorf("error downloading playlist metadata: %w\noutput: %s", err, output)
	}

	fmt.Println(string(output))

	var playlistMetadata PlaylistMetadata
	err = json.Unmarshal(output, &playlistMetadata)
	if err != nil {
		return PlaylistMetadata{}, fmt.Errorf("error parsing playlist metadata: %w\noutput: %s", err, output)
	}
	return playlistMetadata, nil
}
