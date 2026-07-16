package playlist

import (
	"bufio"
	"log/slog"
	"os"
)

func GetPlaylistsFromPlaylistFile(playListFilePath string) []string {
	playlistFile, err := os.Open(playListFilePath)
	if err != nil {
		slog.Error("failed to open playlist file", "path", playListFilePath, "error", err)
		os.Exit(1)
	}
	defer playlistFile.Close()

	reader := bufio.NewReader(playlistFile)
	scanner := bufio.NewScanner(reader)
	var playLists []string
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			playLists = append(playLists, line)
		}
	}
	return playLists
}
