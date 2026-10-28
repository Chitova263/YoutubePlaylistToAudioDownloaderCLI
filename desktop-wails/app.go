package main

import (
	"YoutubePlaylistDownloader/ripper"
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App holds the Wails application state and exposes methods that are bound
// to the JavaScript frontend.
type App struct {
	ctx    context.Context
	mu     sync.Mutex
	cancel context.CancelFunc
}

func NewApp() *App {
	return &App{}
}

// startup is called once by Wails when the application starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.setWorkingDir()
}

// setWorkingDir ensures the process working directory contains a "binaries/"
// folder so that ripper's relative paths (./binaries/yt-dlp.exe, etc.) work.
//
// Resolution order:
//  1. Executable's directory — production: binaries/ is placed next to the .exe
//  2. Current working directory — already correct (e.g. run from project root)
//  3. Parent of cwd — dev: `wails dev` runs from desktop-wails/, binaries/ is one level up
func (a *App) setWorkingDir() {
	hasBinaries := func(dir string) bool {
		_, err := os.Stat(filepath.Join(dir, "binaries"))
		return err == nil
	}

	// 1. Next to the executable (production build)
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if hasBinaries(dir) {
			_ = os.Chdir(dir)
			return
		}
	}

	// 2. Already in a directory that has binaries/ (project root)
	if cwd, err := os.Getwd(); err == nil && hasBinaries(cwd) {
		return
	}

	// 3. Parent directory — wails dev launches from desktop-wails/, binaries/ is at ../
	if cwd, err := os.Getwd(); err == nil {
		parent := filepath.Dir(cwd)
		if hasBinaries(parent) {
			_ = os.Chdir(parent)
			return
		}
	}
}

// shutdown is called by Wails when the application is about to quit.
func (a *App) shutdown(_ context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
	}
}

// DownloadStream starts downloading a YouTube playlist and streams real-time
// events to the frontend. It blocks until the download completes or is
// cancelled; the Promise on the JS side resolves (or rejects) accordingly.
func (a *App) DownloadStream(url, output, format string, parallel, concurrency int) error {
	ctx, cancel := context.WithCancel(a.ctx)
	a.mu.Lock()
	a.cancel = cancel
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.cancel = nil
		a.mu.Unlock()
	}()

	return ripper.Download(ripper.DownloadOptions{
		AudioFormat: format,
		Output:      output,
		Playlists:   []string{url},
		Parallel:    uint(parallel),
		Concurrency: uint(concurrency),
		Context:     ctx,
		Events: ripper.EventHandler{
			OnTrackDiscovered: func(id uint32, title string) {
				runtime.EventsEmit(a.ctx, "track-discovered", map[string]interface{}{
					"id":    id,
					"title": title,
				})
			},
			OnTrackProgress: func(id uint32, percent float32) {
				runtime.EventsEmit(a.ctx, "track-progress", map[string]interface{}{
					"id":      id,
					"percent": percent,
					"status":  "downloading",
				})
			},
			OnTrackComplete: func(id uint32, success bool) {
				runtime.EventsEmit(a.ctx, "track-complete", map[string]interface{}{
					"id":      id,
					"success": success,
				})
			},
			OnDownloadComplete: func(total, failed uint32) {
				code := 0
				if failed > 0 {
					code = 1
				}
				runtime.EventsEmit(a.ctx, "download-complete", map[string]interface{}{
					"code":   code,
					"total":  total,
					"failed": failed,
				})
			},
		},
	})
}

// CancelDownload cancels the active download.
func (a *App) CancelDownload() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
	}
}

// PickFolder opens a native directory-picker dialog and returns the chosen path.
func (a *App) PickFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Output Folder",
	})
}
