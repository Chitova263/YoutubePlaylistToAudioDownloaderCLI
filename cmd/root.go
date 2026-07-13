// Package cmd /*
package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ripper",
	Short: "Download YouTube playlists and convert them to audio",
	Long: `ripper downloads videos from a YouTube playlist and converts them
into audio files (e.g. mp3, wav) for offline listening.

Example:
  ripper rip --url "https://youtube.com/playlist?list=XXXX" --format mp3 --output ./downloads`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	fmt.Println("Root Command Execution")
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Runs after flags have been initialized and before RunE / Run
	cobra.OnInitialize(SetupLogger, InitializeConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ripper.yaml)")
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.YoutubePlaylistDownloader.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func InitializeConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Check default location if no configuration is given
		homeDir, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(homeDir)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".ytpdl") // filename without extension
	}

	// Environment variables
	viper.SetEnvPrefix("ytpdl") // env vars must start with YTPDL_
	viper.AutomaticEnv()        // enable reading from env vars

	if err := viper.ReadInConfig(); err == nil {
		slog.Info("Using config file", "path", viper.ConfigFileUsed())
	}
	// note: if err != nil here, we just proceed with defaults —
	// a missing config file is not fatal
}

func SetupLogger() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)
	slog.Debug("Logger initialized")
}
