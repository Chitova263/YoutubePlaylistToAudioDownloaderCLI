package ytdlp

import (
	"YoutubePlaylistDownloader/playlist"
	"log/slog"
	"path"
	"slices"
	"sync"
	"time"
)

type DownloadOptions struct {
	AudioFormat string
	Output      string
	Input       string
	Playlists   []string
	Parallel    uint
	Concurrency uint
}

func Download(options DownloadOptions) error {

	err := EnsureYtDlpInstalled()
	if err != nil {
		return err
	}

	slog.Debug("Download options", "options", options)

	var playLists []string
	if options.Input != "" {
		playLists = playlist.GetPlaylistsFromPlaylistFile(options.Input)

		if len(playLists) < 1 {
			slog.Info("File has no playlists to download", "file", options.Input)
			return nil
		}
		slog.Info("Extracted playlists from txt file", "playlists", len(playLists))
		slog.Debug("Extracted playlists from file", "playlists", playLists)
	}
	playLists = slices.Concat(playLists, options.Playlists)

	slog.Debug("Playlists", "playlists", playLists)

	playlistMetadataChannel := MetadataExtractionStage(playLists)

	downloadsChannel := DownloadPlaylistItemStage(options.Parallel, playlistMetadataChannel, options.Output)

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
	return nil
}

func DownloadPlaylistItemStage(maxConcurrentDownloads uint, playlistMetadataChannel chan playlist.PlaylistMetadata, downloadsOutputFolderPath string) chan playlist.PlaylistEntry {
	var wg sync.WaitGroup
	downloadsChannel := make(chan playlist.PlaylistEntry, maxConcurrentDownloads)
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
				go func(playlistEntry playlist.PlaylistEntry) {
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

func MetadataExtractionStage(playLists []string) chan playlist.PlaylistMetadata {
	slog.Debug("Playlist metadata extraction", "playlists", playLists)

	var wg sync.WaitGroup
	playlistMetadataChannel := make(chan playlist.PlaylistMetadata, len(playLists))

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

			slog.Info("Playlist metadata extraction success",
				"playlist_id", metadata.ID,
				"playlist_title", metadata.Title,
				"track_count", len(metadata.Entries),
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
