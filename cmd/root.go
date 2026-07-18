package cmd

import (
	"YoutubePlaylistDownloader/logging"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Build-time variables (set via -ldflags)
var version = "dev"

var (
	cfgFile   string
	verbosity int
	logFormat string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ripper [command]",
	Short: "Download YouTube playlists and convert them to audio",
	Long: `ripper downloads videos from a YouTube playlist and converts them
into audio files (e.g. mp3, wav) for offline listening.

Example:
  ripper rip --url "https://youtube.com/playlist?list=XXXX" --format mp3 --output ./downloads`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initLogging, initConfig)

	// Global flags available to all subcommands
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "path to config yaml (default $HOME/.ripper.yaml)")
	_ = rootCmd.MarkPersistentFlagFilename("config", "yaml")

	// Verbosity: -v N
	// Default is 2 (INFO). Users can also use --quiet/-q (sets to 0) or --verbose (sets to 3).
	rootCmd.PersistentFlags().IntVarP(&verbosity, "verbosity", "v", 2, "log verbosity level (0=quiet, 1=warn, 2=info, 3=debug, 4=trace)")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "suppress all output except errors (equivalent to -v 0)")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "enable debug output (equivalent to -v 3)")

	// Log format: text (default, human-readable) or json (machine-parseable for log aggregators)
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log output format: text or json")
}

func initLogging() {
	// Convenience aliases override the numeric verbosity
	if q, _ := rootCmd.PersistentFlags().GetBool("quiet"); q {
		verbosity = 0
	}
	if d, _ := rootCmd.PersistentFlags().GetBool("debug"); d {
		verbosity = 3
	}

	format := logging.FormatText
	if logFormat == "json" {
		format = logging.FormatJSON
	}

	logging.Setup(logging.Config{
		Verbosity: verbosity,
		Format:    format,
		Output:    os.Stderr,
		Component: "ripper",
		Version:   version,
	})

	slog.Debug("logger initialized", "verbosity", verbosity, "format", logFormat)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		homeDir, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(homeDir)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".ripper")
	}

	viper.SetEnvPrefix("RIPPER")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		slog.Info("loaded config file", "path", viper.ConfigFileUsed())
	}
	// Missing config file is not fatal — proceed with defaults + flags
}
