package cmd

import (
	"YoutubePlaylistDownloader/downloader"

	"github.com/spf13/cobra"
)

var ripCmd = &cobra.Command{
	Use:   "rip [flags]",
	Short: "Download and convert a YouTube playlist to audio files",
	Long: `Download every video in a YouTube playlist and convert each track to
the specified audio format for offline listening.

Under the hood, rip uses yt-dlp to fetch video streams and ffmpeg to
extract and encode audio. Downloads run concurrently up to the limit
set by --max-parallel-downloads.

Requires yt-dlp and ffmpeg to be installed and available on PATH.`,
	Example: `  # Download a playlist as mp3 (default)
  ytpdl rip --url "https://youtube.com/playlist?list=PLxxxxxxx"

  # Download as WAV to a custom directory with 4 parallel downloads
  ytpdl rip --url "https://youtube.com/playlist?list=PLxxxxxxx" \
    --format wav --output ~/Music/playlist --max-parallel-downloads 4`,
	Run: func(cmd *cobra.Command, args []string) {
		// Extract flags from cli
		url, _ := cmd.Flags().GetString("url")
		format, _ := cmd.Flags().GetString("format")
		playlist, _ := cmd.Flags().GetString("playlist")
		output, _ := cmd.Flags().GetString("output")
		maxParallelDownloads, _ := cmd.Flags().GetInt("max-parallel-downloads")

		downloader.Download(downloader.DownloadOptions{
			Url:                  url,
			Format:               format,
			Output:               output,
			MaxParallelDownloads: maxParallelDownloads,
			Playlists:            playlist,
		})
	},
}

func init() {
	rootCmd.AddCommand(ripCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// ripCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly
	ripCmd.Flags().String("url", "", "YouTube playlist url")
	ripCmd.Flags().String("format", "mp3", "Output format")
	ripCmd.Flags().String("output", ".", "Output directory")
	ripCmd.Flags().String("playlist", "", "Playlist to download")
	ripCmd.Flags().Int("max-parallel-downloads", 2, "Maximum number of Parallel Downloads")
}
