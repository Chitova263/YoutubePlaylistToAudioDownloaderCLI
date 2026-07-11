package main

import (
	"fmt"
	"os"
)

func main() {
	err := DownloadYtDlpStandaloneBinary("binaries")
	if err != nil {
		panic(err)
	}

	playListUrl := "https://music.youtube.com/watch?v=OQngNsJRAeA&list=PLIludI4VH2HFdfMfUmGnYZ9uTVTwurF1O"

	playlistMetadataDownloadOptions := PlaylistMetadataDownloadOptions{
		PlayListUrl:             playListUrl,
		SkipPlayListAfterErrors: "5",
	}

	// Download playlist metadata
	metadata, err := DownloadPlaylistMetadata(playlistMetadataDownloadOptions)
	if err != nil {
		os.Exit(1)
	}

	fmt.Println(metadata)

	// Download playlist
	fmt.Println("Downloading playlist...")
	options := PlaylistDownloadOption{
		AudioFormat:         FormatMP3,
		OutputFolderPath:    "./output",
		ConcurrentFragments: 4,
		Thumbnail:           false,
		PlayListUrl:         playListUrl,
		FormatSelector:      "bestaudio/best",
		AudioQuality:        "0",
	}

	err = DownloadPlaylist(options)
	if err != nil {
		os.Exit(1)
	}
}
