package ripper

// EventHandler holds optional callback functions invoked during a download
// pipeline. All fields are optional; nil callbacks are silently skipped.
// This lets the CLI use DownloadOptions with zero callbacks while the GUI
// receives real-time progress updates without any log-line parsing.
type EventHandler struct {
	// OnTrackDiscovered is called just before a track download begins.
	OnTrackDiscovered func(id uint32, title string)
	// OnTrackProgress is called periodically with the current download
	// percentage (0–100).
	OnTrackProgress func(id uint32, percent float32)
	// OnTrackComplete is called when a track finishes. success is false on error.
	OnTrackComplete func(id uint32, success bool)
	// OnDownloadComplete is called once every track has been processed.
	OnDownloadComplete func(total, failed uint32)
}
