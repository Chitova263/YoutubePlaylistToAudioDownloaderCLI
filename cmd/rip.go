package cmd

import (
	"YoutubePlaylistDownloader/downloader"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
  ripper rip --url "https://youtube.com/playlist?list=PLxxxxxxx"

  # Download as WAV to a custom directory with 4 parallel downloads
  ripper rip --url "https://youtube.com/playlist?list=PLxxxxxxx" \
    --format wav --output ~/Music/playlist --max-parallel-downloads 4`,
	RunE: func(cmd *cobra.Command, args []string) error {

		url, err := cmd.Flags().GetString("url")
		cobra.CheckErr(err)

		format := viper.GetString("format")
		playlist := viper.GetString("playlist")
		output := viper.GetString("output")
		maxParallelDownloads := viper.GetInt("max-parallel-downloads")

		slog.Debug("Flags", "format", format, "playlist", playlist, "output", output, "max-parallel-downloads", maxParallelDownloads, "url", url)

		downloader.Download(downloader.DownloadOptions{
			Url:                  url,
			Format:               format,
			Output:               output,
			MaxParallelDownloads: maxParallelDownloads,
			Playlists:            playlist,
		})
		return nil
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
	format := "format"
	ripCmd.Flags().String(format, "mp3", "Output format")
	output := "output"
	ripCmd.Flags().String(output, ".", "Output directory")
	playlist := "playlist"
	ripCmd.Flags().String(playlist, "", "Playlist to download")
	maxParallelDownloads := "max-parallel-downloads"
	ripCmd.Flags().Int(maxParallelDownloads, 2, "Maximum number of Parallel Downloads")

	// Bind flags to viper configuration manager
	err := viper.BindPFlag(format, ripCmd.Flags().Lookup(format))
	cobra.CheckErr(err)
	err = viper.BindPFlag(output, ripCmd.Flags().Lookup(output))
	cobra.CheckErr(err)
	err = viper.BindPFlag(playlist, ripCmd.Flags().Lookup(playlist))
	cobra.CheckErr(err)
	err = viper.BindPFlag(maxParallelDownloads, ripCmd.Flags().Lookup(maxParallelDownloads))
	cobra.CheckErr(err)
}
