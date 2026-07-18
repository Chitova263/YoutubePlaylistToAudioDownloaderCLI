package ripper

import (
	"YoutubePlaylistDownloader/logging"
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// installLog is the component logger for dependency installation.
var installLog = logging.Component("install")

type BinaryDownloadOption struct {
	URL      string
	Filename string
}

func EnsureYtDlpInstalled() error {
	platformBinaries := map[string]BinaryDownloadOption{
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

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating binaries directory: %w", err)
	}

	if _, err := os.Stat(pathToFile); err == nil {
		installLog.Debug("yt-dlp already present", "path", pathToFile)
		return nil
	}

	installLog.Info("downloading yt-dlp",
		"url", ytDlpDownloadOption.URL,
		"destination", pathToFile,
		"platform", runtime.GOOS,
	)

	response, err := http.Get(ytDlpDownloadOption.URL)
	if err != nil {
		return fmt.Errorf("downloading yt-dlp: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if err := os.WriteFile(pathToFile, body, 0755); err != nil {
		return fmt.Errorf("writing %s: %w", pathToFile, err)
	}

	installLog.Info("yt-dlp installed successfully", "path", pathToFile)
	return nil
}

// FFmpegDownloadOption holds the URL and expected binary names for a platform's ffmpeg build.
type FFmpegDownloadOption struct {
	URL         string
	Binaries    []string // e.g. ["ffmpeg.exe", "ffprobe.exe"] or ["ffmpeg", "ffprobe"]
	ArchiveType string   // "zip" or "tar.xz"
}

func EnsureFFmpegInstalled() error {
	platformBuilds := map[string]FFmpegDownloadOption{
		"windows": {
			URL:         "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip",
			Binaries:    []string{"ffmpeg.exe", "ffprobe.exe"},
			ArchiveType: "zip",
		},
		"linux": {
			URL:         "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-linux64-gpl.tar.xz",
			Binaries:    []string{"ffmpeg", "ffprobe"},
			ArchiveType: "tar.xz",
		},
		"darwin": {
			URL:         "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-linux64-gpl.tar.xz",
			Binaries:    []string{"ffmpeg", "ffprobe"},
			ArchiveType: "tar.xz",
		},
	}

	build, exists := platformBuilds[runtime.GOOS]
	if !exists {
		return fmt.Errorf("unsupported platform for ffmpeg: %s", runtime.GOOS)
	}

	binDir := "binaries"
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("creating binaries directory: %w", err)
	}

	// Check if all required binaries already exist
	allPresent := true
	for _, bin := range build.Binaries {
		path := filepath.Join(binDir, bin)
		if _, err := os.Stat(path); err != nil {
			allPresent = false
			break
		}
	}
	if allPresent {
		installLog.Debug("ffmpeg already present", "path", binDir)
		return nil
	}

	installLog.Info("downloading ffmpeg",
		"url", build.URL,
		"destination", binDir,
		"platform", runtime.GOOS,
	)

	// Download the archive
	response, err := http.Get(build.URL)
	if err != nil {
		return fmt.Errorf("downloading ffmpeg: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("ffmpeg download failed: HTTP %d", response.StatusCode)
	}

	// Save to temp file
	tmpFile, err := os.CreateTemp("", "ffmpeg-*."+build.ArchiveType)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, response.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("saving ffmpeg archive: %w", err)
	}
	tmpFile.Close()

	installLog.Info("extracting ffmpeg binaries", "archive", tmpPath)

	switch build.ArchiveType {
	case "zip":
		if err := extractFFmpegFromZip(tmpPath, binDir, build.Binaries); err != nil {
			return fmt.Errorf("extracting ffmpeg from zip: %w", err)
		}
	case "tar.xz":
		if err := extractFFmpegFromTarXz(tmpPath, binDir, build.Binaries); err != nil {
			return fmt.Errorf("extracting ffmpeg from tar.xz: %w", err)
		}
	default:
		return fmt.Errorf("unsupported archive type: %s", build.ArchiveType)
	}

	installLog.Info("ffmpeg installed successfully", "path", binDir)
	return nil
}

// extractFFmpegFromZip extracts the target binaries from a zip archive.
// The BtbN builds have structure: ffmpeg-master-latest-win64-gpl/bin/ffmpeg.exe
func extractFFmpegFromZip(zipPath, destDir string, binaries []string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	binSet := make(map[string]bool)
	for _, b := range binaries {
		binSet[b] = true
	}

	extracted := 0
	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if !binSet[name] {
			continue
		}
		// Only extract from the bin/ directory
		if !strings.Contains(f.Name, "/bin/") && !strings.Contains(f.Name, "\\bin\\") {
			continue
		}

		destPath := filepath.Join(destDir, name)
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("opening %s in archive: %w", f.Name, err)
		}

		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			rc.Close()
			return fmt.Errorf("creating %s: %w", destPath, err)
		}

		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return fmt.Errorf("writing %s: %w", destPath, err)
		}

		out.Close()
		rc.Close()
		extracted++
		installLog.Debug("extracted", "file", destPath)
	}

	if extracted == 0 {
		return fmt.Errorf("no ffmpeg binaries found in archive")
	}
	return nil
}

// extractFFmpegFromTarXz extracts the target binaries from a tar.xz archive.
func extractFFmpegFromTarXz(archivePath, destDir string, binaries []string) error {
	// We shell out to tar since Go has no built-in xz support
	for _, bin := range binaries {
		// Extract with tar, using --wildcards to find the bin/ files
		cmd := exec.Command("tar", "-xf", archivePath, "--wildcards", "*/bin/"+bin, "--strip-components=2", "-C", destDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("extracting %s: %w\noutput: %s", bin, err, string(output))
		}
		// Ensure executable permission
		os.Chmod(filepath.Join(destDir, bin), 0755)
	}
	return nil
}
