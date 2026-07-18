package playlist

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
)

// GetPlaylistsFromPlaylistFile reads playlist URLs from a text file (one per line).
// Empty lines are skipped.
func GetPlaylistsFromPlaylistFile(filePath string) []string {
	log := slog.Default().With("component", "playlist")

	log.Debug("reading playlist file", "path", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		// Fatal: if the user specified --input, the file must exist
		log.Error("failed to open playlist file", "path", filePath, "error", err)
		fmt.Fprintf(os.Stderr, "Error: cannot open playlist file %q: %v\n", filePath, err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(bufio.NewReader(file))
	var playlists []string
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line != "" {
			playlists = append(playlists, line)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Error("error reading playlist file", "path", filePath, "error", err)
	}

	log.Debug("playlist file parsed", "path", filePath, "lines_read", lineNum, "urls_found", len(playlists))
	return playlists
}
