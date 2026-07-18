package cmd

import (
	"YoutubePlaylistDownloader/ripper"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var ripCmd = &cobra.Command{
	Use:   "rip (--input FILE | --playlists URLS) [flags]",
	Short: "Download and convert a YouTube playlist to audio files",
	Long: `Download every video in a YouTube playlist and convert each track to
the specified audio format for offline listening.

Under the hood, rip uses yt-dlp to fetch video streams and ffmpeg to
extract and encode audio. Playlist downloads run in parallel up to the
limit set by --parallel, and within each video, stream fragments are
fetched concurrently up to the limit set by --concurrency.

Exactly one of --input or --playlists must be specified to select which
playlist(s) to download; they are mutually exclusive.

Both ffmpeg and yt-dlp are fetched automatically on first run.`,
	Example: `  # Download a single playlist as mp3
  ripper rip --playlists "https://youtube.com/playlist?list=PLxxxxxxx" \
    --audio-format mp3 --output ~/Music/playlist

  # Download multiple playlists from a text file, 4 in parallel
  ripper rip --input playlists.txt \
    --audio-format wav --output ~/Music/playlist --parallel 4`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return validateFlags()
	},

	RunE: func(cmd *cobra.Command, args []string) error {
		flags := getRipCommandFlags()

		slog.Info("starting rip",
			"audio_format", flags.AudioFormat,
			"output", flags.Output,
			"parallel", flags.Parallel,
			"concurrency", flags.Concurrency,
		)

		err := ripper.Download(ripper.DownloadOptions{
			AudioFormat: flags.AudioFormat,
			Output:      flags.Output,
			Input:       flags.Input,
			Playlists:   splitPlaylists(flags.Playlists),
			Parallel:    flags.Parallel,
			Concurrency: flags.Concurrency,
		})
		if err != nil {
			slog.Error("rip failed", "error", err)
			return err
		}

		slog.Info("rip completed successfully")
		return nil
	},
}

func splitPlaylists(playlists string) []string {
	if playlists != "" {
		return strings.Split(playlists, ",")
	}
	return nil
}

func validateFlags() error {
	flags := getRipCommandFlags()

	allowedFormats := []string{"mp3", "wav", "m4a", "opus", "flac"}
	if !slices.Contains(allowedFormats, flags.AudioFormat) {
		return fmt.Errorf("invalid audio format %q (allowed: %s)", flags.AudioFormat, strings.Join(allowedFormats, ", "))
	}

	if flags.Parallel < 1 {
		return fmt.Errorf("--parallel must be at least 1, got %d", flags.Parallel)
	}

	if flags.Concurrency < 1 {
		return fmt.Errorf("--concurrency must be at least 1, got %d", flags.Concurrency)
	}

	slog.Debug("flags validated",
		"audio_format", flags.AudioFormat,
		"parallel", flags.Parallel,
		"concurrency", flags.Concurrency,
	)
	return nil
}

func init() {
	rootCmd.AddCommand(ripCmd)

	ripCmd.Flags().StringP("audio-format", "", "", "output audio format (mp3, wav, m4a, opus, flac)")
	_ = ripCmd.MarkFlagRequired("audio-format")

	ripCmd.Flags().StringP("output", "o", "", "output directory")
	_ = ripCmd.MarkFlagRequired("output")

	ripCmd.Flags().StringP("input", "i", "", "text file with YouTube playlist URLs (one per line)")
	ripCmd.Flags().String("playlists", "", "comma-separated YouTube playlist URLs")

	ripCmd.MarkFlagsMutuallyExclusive("playlists", "input")
	ripCmd.MarkFlagsOneRequired("playlists", "input")

	ripCmd.Flags().Uint("parallel", 2, "maximum number of parallel downloads")
	ripCmd.Flags().Uint("concurrency", 3, "concurrent fragments per video download")

	// Bind flags to viper
	for _, name := range []string{"audio-format", "output", "input", "playlists", "parallel", "concurrency"} {
		_ = viper.BindPFlag(name, ripCmd.Flags().Lookup(name))
	}
}

type ripCommandFlags struct {
	AudioFormat string `mapstructure:"audio-format"`
	Output      string `mapstructure:"output"`
	Input       string `mapstructure:"input"`
	Playlists   string `mapstructure:"playlists"`
	Parallel    uint   `mapstructure:"parallel"`
	Concurrency uint   `mapstructure:"concurrency"`
}

func getRipCommandFlags() ripCommandFlags {
	var flags ripCommandFlags
	err := viper.Unmarshal(&flags)
	cobra.CheckErr(err)
	return flags
}
