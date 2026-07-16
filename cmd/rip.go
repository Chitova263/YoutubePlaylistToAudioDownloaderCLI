package cmd

import (
	"YoutubePlaylistDownloader/ytdlp"
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

Requires yt-dlp and ffmpeg to be installed and available on PATH.`,
	Example: `  # Download a single playlist as mp3
  ripper rip --playlists "https://youtube.com/playlist?list=PLxxxxxxx" \
    --audio-format mp3 --output ~/Music/playlist

  # Download multiple playlists from a text file, 4 in parallel
  ripper rip --input playlists.txt \
    --audio-format wav --output ~/Music/playlist --parallel 4`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return ValidateFlags()
	},

	RunE: func(cmd *cobra.Command, args []string) error {
		flags := GetRipCommandFlags()
		err := ytdlp.Download(ytdlp.DownloadOptions{
			AudioFormat: flags.AudioFormat,
			Output:      flags.Output,
			Input:       flags.Input,
			Playlists:   getPlaylists(flags.Playlists),
			Parallel:    flags.Parallel,
			Concurrency: flags.Concurrency,
		})
		if err != nil {
			slog.Error("Error Exiting", "error", err)
			return err
		}
		return nil
	},
}

func getPlaylists(playlists string) []string {
	if playlists != "" {
		return strings.Split(playlists, ",")
	}
	return make([]string, 0)
}

func ValidateFlags() error {
	flags := GetRipCommandFlags()

	allowedOutputAudioFormats := make([]string, 5)
	allowedOutputAudioFormats = append(allowedOutputAudioFormats, "mp3", "wav")
	if !slices.Contains(allowedOutputAudioFormats, flags.AudioFormat) {
		return fmt.Errorf("invalid audio format %s", flags.AudioFormat)
	}

	if flags.Parallel < 1 {
		return fmt.Errorf("--parallel cannot be less than 1 %d ", flags.Parallel)
	}

	if flags.Concurrency < 1 {
		return fmt.Errorf("--concurrency cannot be less than 1 %d ", flags.Concurrency)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(ripCmd)
	// Here you will define your flags and configuration settings.

	ripCmd.Flags().StringP("audio-format", "", "", "output audio format")
	err := ripCmd.MarkFlagRequired("audio-format")
	cobra.CheckErr(err)

	ripCmd.Flags().StringP("output", "o", "", "output directory")
	err = ripCmd.MarkFlagRequired("output")
	cobra.CheckErr(err)

	ripCmd.Flags().StringP("input", "i", "", "text file with YouTube playlist URLs to download (mutually exclusive with --playlists)")
	ripCmd.Flags().String("playlists", "", "comma separated YouTube playlist URLs to download (mutually exclusive with --input)")
	err = ripCmd.MarkFlagFilename("playlists", ".txt")
	cobra.CheckErr(err)

	ripCmd.MarkFlagsMutuallyExclusive("playlists", "input")
	ripCmd.MarkFlagsOneRequired("playlists", "input")

	ripCmd.Flags().Int("parallel", 2, "maximum number of parallel downloads")

	ripCmd.Flags().Int("concurrency", 3, "number of fragments of a dash/hlsnative video that should be downloaded concurrently")

	// Bind flags to viper configuration manager
	err = viper.BindPFlag("audio-format", ripCmd.Flags().Lookup("audio-format"))
	cobra.CheckErr(err)
	err = viper.BindPFlag("output", ripCmd.Flags().Lookup("output"))
	cobra.CheckErr(err)
	err = viper.BindPFlag("input", ripCmd.Flags().Lookup("input"))
	cobra.CheckErr(err)
	err = viper.BindPFlag("playlists", ripCmd.Flags().Lookup("playlists"))
	cobra.CheckErr(err)
	err = viper.BindPFlag("parallel", ripCmd.Flags().Lookup("parallel"))
	cobra.CheckErr(err)
	err = viper.BindPFlag("concurrency", ripCmd.Flags().Lookup("concurrency"))
	cobra.CheckErr(err)
}

type RipCommandFlags struct {
	AudioFormat string `mapstructure:"audio-format"`
	Output      string `mapstructure:"output"`
	Input       string `mapstructure:"input"`
	Playlists   string `mapstructure:"playlists"`
	Parallel    uint   `mapstructure:"parallel"`
	Concurrency uint   `mapstructure:"concurrency"`
}

func GetRipCommandFlags() RipCommandFlags {
	var flags RipCommandFlags
	err := viper.Unmarshal(&flags)
	cobra.CheckErr(err)
	return flags
}
