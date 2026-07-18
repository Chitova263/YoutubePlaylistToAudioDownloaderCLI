package ripper

import (
	"YoutubePlaylistDownloader/logging"
	"YoutubePlaylistDownloader/playlist"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"sync"
	"syscall"
	"time"
)

// log is the component logger for the pipeline package.
// Every log line from this package carries component=pipeline.
var log = logging.Component("pipeline")

type DownloadOptions struct {
	AudioFormat string
	Output      string
	Input       string
	Playlists   []string
	Parallel    uint
	Concurrency uint
}

func Download(options DownloadOptions) error {
	log.Info("starting download pipeline",
		"audio_format", options.AudioFormat,
		"output_dir", options.Output,
		"parallel", options.Parallel,
	)

	// Set up graceful shutdown: cancel context on SIGINT/SIGTERM.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Warn("received shutdown signal, finishing in-flight downloads...", "signal", sig)
		cancel()
	}()

	err := EnsureYtDlpInstalled()
	if err != nil {
		return err
	}

	err = EnsureFFmpegInstalled()
	if err != nil {
		return err
	}

	logging.Trace("download options resolved", "options", options)

	var playLists []string
	if options.Input != "" {
		playLists = playlist.GetPlaylistsFromPlaylistFile(options.Input)

		if len(playLists) < 1 {
			log.Warn("input file contains no playlist URLs", "file", options.Input)
			return nil
		}
		log.Info("loaded playlists from file", "file", options.Input, "count", len(playLists))
		log.Debug("playlist URLs from file", "urls", playLists)
	}
	playLists = slices.Concat(playLists, options.Playlists)

	log.Info("total playlists to process", "count", len(playLists))
	logging.Trace("all playlist URLs", "urls", playLists)

	playlistMetadataChannel := MetadataExtractionStage(ctx, playLists)
	downloadsChannel := DownloadPlaylistItemStage(ctx, options, playlistMetadataChannel)

	completed := 0
	for entry := range downloadsChannel {
		completed++
		log.Info("track download complete",
			"video_id", entry.ID,
			"title", entry.Title,
			"total_completed", completed,
		)
	}

	if ctx.Err() != nil {
		log.Warn("pipeline was interrupted", "total_completed", completed)
		return fmt.Errorf("interrupted: completed %d downloads before shutdown", completed)
	}

	log.Info("all downloads finished", "total_tracks", completed)
	return nil
}

func DownloadPlaylistItemStage(ctx context.Context, opts DownloadOptions, playlistMetadataChannel chan playlist.PlaylistMetadata) chan playlist.PlaylistEntry {
	var wg sync.WaitGroup
	downloadsChannel := make(chan playlist.PlaylistEntry, opts.Parallel)
	sem := make(chan struct{}, opts.Parallel)

	const maxRetries = 3

	go func() {
		for metadata := range playlistMetadataChannel {
			// Check for cancellation between playlists
			if ctx.Err() != nil {
				log.Debug("skipping playlist due to shutdown", "playlist_id", metadata.ID)
				break
			}

			sanitizedPlaylistTitle := SanitizePath(metadata.Title)
			trackCount := len(metadata.Entries)

			log.Info("processing playlist",
				"playlist_id", metadata.ID,
				"playlist_title", metadata.Title,
				"track_count", trackCount,
			)

			// Create output directory before any downloads start.
			playlistDir := filepath.Join(opts.Output, sanitizedPlaylistTitle)
			if err := os.MkdirAll(playlistDir, 0o755); err != nil {
				log.Error("failed to create playlist directory",
					"path", playlistDir,
					"error", err,
				)
				continue
			}

			// Each playlist gets its own archive file so yt-dlp skips
			// already-downloaded videos on retry/restart.
			// Safe for concurrent use: each goroutine downloads a unique video ID,
			// so there's no risk of duplicate work. Appends of short lines (one
			// archive entry like "youtube dQw4w9WgXcQ\n") are effectively atomic.
			archivePath := filepath.Join(playlistDir, ".download-archive")

			for i, entry := range metadata.Entries {
				// Check for cancellation between tracks
				if ctx.Err() != nil {
					log.Debug("stopping track dispatch due to shutdown", "playlist_id", metadata.ID)
					break
				}

				trackNum := i + 1
				wg.Add(1)
				sem <- struct{}{} // acquire semaphore slot

				go func(playlistEntry playlist.PlaylistEntry, num int) {
					defer wg.Done()
					defer func() { <-sem }() // release slot

					// Check for cancellation before starting download
					if ctx.Err() != nil {
						return
					}

					// Use yt-dlp's %(title)s placeholder for file naming.
					// This lets yt-dlp sanitize the filename itself and avoids
					// baking the title (which may contain illegal chars) into the path.
					outputTemplate := filepath.Join(playlistDir, "%(title)s.%(ext)s")

					dlOpt := DownloadOption{
						AudioFormat:         AudioFormat(opts.AudioFormat),
						OutputFolderPath:    outputTemplate,
						ConcurrentFragments: opts.Concurrency,
						Thumbnail:           true,
						Url:                 playlistEntry.URL,
						FormatSelector:      "bestaudio/best",
						AudioQuality:        "0",
						ArchivePath:         archivePath,
					}

					log.Info("downloading track",
						"playlist_id", metadata.ID,
						"progress", fmt.Sprintf("%d/%d", num, trackCount),
						"video_id", playlistEntry.ID,
						"video_title", playlistEntry.Title,
					)

					start := time.Now()

					var err error
					for attempt := 1; attempt <= maxRetries; attempt++ {
						// Check for cancellation before each retry
						if ctx.Err() != nil {
							return
						}

						err = DownloadPlaylist(dlOpt)

						if err == nil {
							break
						}
						if attempt < maxRetries {
							backoff := time.Duration(attempt*2) * time.Second
							log.Warn("download attempt failed, retrying",
								"playlist_id", metadata.ID,
								"video_id", playlistEntry.ID,
								"video_title", playlistEntry.Title,
								"attempt", attempt,
								"backoff", backoff,
								"error", err,
							)
							time.Sleep(backoff)
						}
					}

					duration := time.Since(start)

					if err != nil {
						log.Error("download failed after retries",
							"playlist_id", metadata.ID,
							"video_id", playlistEntry.ID,
							"video_title", playlistEntry.Title,
							"progress", fmt.Sprintf("%d/%d", num, trackCount),
							"attempts", maxRetries,
							"duration", duration,
							"error", err,
						)
						return
					}

					log.Info("download succeeded",
						"playlist_id", metadata.ID,
						"video_id", playlistEntry.ID,
						"video_title", playlistEntry.Title,
						"progress", fmt.Sprintf("%d/%d", num, trackCount),
						"duration", duration,
					)
					downloadsChannel <- playlistEntry

				}(entry, trackNum)
			}
		}
		wg.Wait()
		close(downloadsChannel)
	}()
	return downloadsChannel
}

func MetadataExtractionStage(ctx context.Context, playLists []string) chan playlist.PlaylistMetadata {
	log.Debug("starting metadata extraction", "playlist_count", len(playLists))

	var wg sync.WaitGroup
	playlistMetadataChannel := make(chan playlist.PlaylistMetadata, len(playLists))

	for _, playListDownloadUrl := range playLists {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			// Check for cancellation before starting
			if ctx.Err() != nil {
				return
			}

			log.Debug("extracting metadata", "url", url)

			start := time.Now()
			metadata, err := DownloadPlaylistMetadata(PlaylistMetadataDownloadOptions{
				PlayListUrl: url,
			})
			duration := time.Since(start)

			if err != nil {
				log.Error("metadata extraction failed",
					"url", url,
					"duration", duration,
					"error", err,
				)
				return
			}

			playlistMetadataChannel <- metadata

			log.Info("metadata extracted",
				"playlist_id", metadata.ID,
				"playlist_title", metadata.Title,
				"track_count", len(metadata.Entries),
				"duration", duration,
			)
		}(playListDownloadUrl)
	}

	go func() {
		wg.Wait()
		close(playlistMetadataChannel)
	}()
	return playlistMetadataChannel
}
