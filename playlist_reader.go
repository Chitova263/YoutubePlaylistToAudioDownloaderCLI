package main

import (
	"bufio"
	"fmt"
	"os"
)

func GetPlaylistsFromPlaylistFile(playListFilePath string) []string {
	playlistFile, err := os.Open(playListFilePath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer playlistFile.Close()

	reader := bufio.NewReader(playlistFile)
	scanner := bufio.NewScanner(reader)
	var playLists []string
	for scanner.Scan() {
		playLists = append(playLists, scanner.Text())
	}
	return playLists
}
