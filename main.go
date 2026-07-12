package main

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"sync"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	err := EnsureYtDlpInstalled()
	if err != nil {
		os.Exit(1)
	}

	playListFilePath := "playlist.txt"
	downloadsOutputFolderPath := "./output"
	maxConcurrentDownloads := 5
	playLists := GetPlaylistsFromPlaylistFile(playListFilePath)
	playlistMetadataChannel := MetadataExtractionStage(playLists)
	downloadsChannel := DownloadPlaylistItemStage(maxConcurrentDownloads, playlistMetadataChannel, downloadsOutputFolderPath)

	for entry := range downloadsChannel {
		fmt.Printf("Downloaded %s\n", entry.URL)
	}
}

func DownloadPlaylistItemStage(maxConcurrentDownloads int, playlistMetadataChannel chan PlaylistMetadata, downloadsOutputFolderPath string) chan PlaylistEntry {
	var wg sync.WaitGroup
	downloadsChannel := make(chan PlaylistEntry, maxConcurrentDownloads)
	sem := make(chan struct{}, maxConcurrentDownloads)
	go func() {
		for metadata := range playlistMetadataChannel {
			for _, entry := range metadata.Entries {
				wg.Add(1)
				sem <- struct{}{} // acquire slot, blocks if max concurrent reached
				go func(playlistEntry PlaylistEntry) {
					defer wg.Done()
					defer func() { <-sem }() // release slot
					options := DownloadOption{
						AudioFormat:         FormatMP3,
						OutputFolderPath:    path.Join(downloadsOutputFolderPath, metadata.Title, playlistEntry.Title),
						ConcurrentFragments: 16,
						Thumbnail:           false,
						Url:                 playlistEntry.URL,
						FormatSelector:      "bestaudio/best",
						AudioQuality:        "0",
					}

					slog.Info("Downloading playlist item", "Playlist Id", metadata.ID, "Playlist Title", metadata.Title, "ItemId", playlistEntry.ID, "ItemTitle", playlistEntry.Title)

					err := DownloadPlaylist(options)
					if err != nil {
						slog.Error("Error Playlist Metadata", "Title", metadata.Title, "VideoCount", len(metadata.Entries), err.Error())
						return
					}
					downloadsChannel <- playlistEntry
					slog.Info("Downloaded playlist item", "Playlist Id", metadata.ID, "Playlist Title", metadata.Title, "ItemId", playlistEntry.ID, "ItemTitle", playlistEntry.Title)

				}(entry)
			}
		}
		wg.Wait()
		close(downloadsChannel)
	}()
	return downloadsChannel
}

func MetadataExtractionStage(playLists []string) chan PlaylistMetadata {
	var wg sync.WaitGroup
	playlistMetadataChannel := make(chan PlaylistMetadata, len(playLists))
	for _, playListDownloadUrl := range playLists {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			metadata, err := DownloadPlaylistMetadata(PlaylistMetadataDownloadOptions{
				PlayListUrl: url,
			})
			if err != nil {
				fmt.Println(err)
			}
			playlistMetadataChannel <- metadata
			slog.Info("Downloaded Playlist Metadata", "Title", metadata.Title, "VideoCount", len(metadata.Entries), "Playlist URL", url)

		}(playListDownloadUrl)
	}

	go func() {
		wg.Wait()
		close(playlistMetadataChannel)
	}()
	return playlistMetadataChannel
}
