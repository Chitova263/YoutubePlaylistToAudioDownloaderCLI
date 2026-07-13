package downloader

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

type YtDlpBinaryDownloadOption struct {
	URL      string
	Filename string
}

func EnsureYtDlpInstalled() error {
	platformBinaries := map[string]YtDlpBinaryDownloadOption{
		"windows": {
			URL:      "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe",
			Filename: "yt-dlp.exe",
		},
		"linux": {
			URL:      "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp",
			Filename: "yt-dlp",
		},
		"darwin": {
			URL:      "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_macos",
			Filename: "yt-dlp",
		},
	}

	ytDlpDownloadOption, exists := platformBinaries[runtime.GOOS]
	if !exists {
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	pathToFile := filepath.Join("binaries", ytDlpDownloadOption.Filename)
	dir := filepath.Dir(pathToFile)

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	_, err = os.Stat(pathToFile)
	if err == nil {
		slog.Info("yt-dlp binary already installed", "path", pathToFile)
		return nil
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check %s: %w", pathToFile, err)
	}

	slog.Info("downloading yt-dlp binary", "url", ytDlpDownloadOption.URL, "destination", pathToFile)

	response, err := http.Get(ytDlpDownloadOption.URL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: unexpected status %d", response.StatusCode)
	}

	responseBodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	err = os.WriteFile(pathToFile, responseBodyBytes, 0755)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", pathToFile, err)
	}

	slog.Info("yt-dlp binary installed successfully", "path", pathToFile)
	return nil
}
