package downloader

import (
	"log/slog"
	"os"
	"path"
	"sync"
	"time"
)

type DownloadOptions struct {
	Url                  string
	Format               string
	Output               string
	MaxParallelDownloads int
	Playlists            string
}

func Download(options DownloadOptions) {

	err := EnsureYtDlpInstalled()
	if err != nil {
		slog.Error("failed to ensure yt-dlp is installed", "error", err)
		os.Exit(1)
	}

	slog.Info("starting playlist downloader",
		"playlist_file", options.Playlists,
		"output_dir", options.Output,
		"max_concurrent_downloads", options.MaxParallelDownloads,
	)

	playLists := GetPlaylistsFromPlaylistFile(options.Playlists)
	slog.Info("loaded playlists", "count", len(playLists))

	playlistMetadataChannel := MetadataExtractionStage(playLists)
	downloadsChannel := DownloadPlaylistItemStage(options.MaxParallelDownloads, playlistMetadataChannel, options.Output)

	completed := 0
	for entry := range downloadsChannel {
		completed++
		slog.Info("download complete",
			"video_id", entry.ID,
			"title", entry.Title,
			"url", entry.URL,
			"total_completed", completed,
		)
	}

	slog.Info("all downloads finished", "total_completed", completed)
}

func DownloadPlaylistItemStage(maxConcurrentDownloads int, playlistMetadataChannel chan PlaylistMetadata, downloadsOutputFolderPath string) chan PlaylistEntry {
	var wg sync.WaitGroup
	downloadsChannel := make(chan PlaylistEntry, maxConcurrentDownloads)
	sem := make(chan struct{}, maxConcurrentDownloads)

	go func() {
		for metadata := range playlistMetadataChannel {
			slog.Info("processing playlist",
				"playlist_id", metadata.ID,
				"playlist_title", metadata.Title,
				"track_count", len(metadata.Entries),
			)

			for _, entry := range metadata.Entries {
				wg.Add(1)
				sem <- struct{}{} // acquire slot, blocks if max concurrent reached
				go func(playlistEntry PlaylistEntry) {
					defer wg.Done()
					defer func() { <-sem }() // release slot

					options := DownloadOption{
						AudioFormat:         FormatMP3,
						OutputFolderPath:    path.Join(downloadsOutputFolderPath, metadata.Title, playlistEntry.Title) + "%(ext)s",
						ConcurrentFragments: 16,
						Thumbnail:           false,
						Url:                 playlistEntry.URL,
						FormatSelector:      "bestaudio/best",
						AudioQuality:        "0",
					}

					slog.Info("download started",
						"playlist_id", metadata.ID,
						"playlist_title", metadata.Title,
						"video_id", playlistEntry.ID,
						"video_title", playlistEntry.Title,
					)

					start := time.Now()
					err := DownloadPlaylist(options)
					duration := time.Since(start)

					if err != nil {
						slog.Error("download failed",
							"playlist_id", metadata.ID,
							"playlist_title", metadata.Title,
							"video_id", playlistEntry.ID,
							"video_title", playlistEntry.Title,
							"duration_ms", duration.Milliseconds(),
							"error", err,
						)
						return
					}

					slog.Info("download succeeded",
						"playlist_id", metadata.ID,
						"playlist_title", metadata.Title,
						"video_id", playlistEntry.ID,
						"video_title", playlistEntry.Title,
						"duration_ms", duration.Milliseconds(),
					)
					downloadsChannel <- playlistEntry

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

			start := time.Now()
			metadata, err := DownloadPlaylistMetadata(PlaylistMetadataDownloadOptions{
				PlayListUrl: url,
			})
			duration := time.Since(start)

			if err != nil {
				slog.Error("metadata extraction failed",
					"playlist_url", url,
					"duration_ms", duration.Milliseconds(),
					"error", err,
				)
				return
			}

			playlistMetadataChannel <- metadata
			slog.Info("metadata extraction succeeded",
				"playlist_id", metadata.ID,
				"playlist_title", metadata.Title,
				"track_count", len(metadata.Entries),
				"playlist_url", url,
				"duration_ms", duration.Milliseconds(),
			)
		}(playListDownloadUrl)
	}

	go func() {
		wg.Wait()
		close(playlistMetadataChannel)
	}()
	return playlistMetadataChannel
}
