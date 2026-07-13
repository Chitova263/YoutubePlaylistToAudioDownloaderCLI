package downloader

// Thumbnail represents a video/playlist thumbnail at a specific resolution.
type Thumbnail struct {
	URL        string `json:"url"`
	Height     int    `json:"height"`
	Width      int    `json:"width"`
	ID         string `json:"id,omitempty"`
	Resolution string `json:"resolution,omitempty"`
}

// PlaylistEntry represents a single video entry within a playlist.
type PlaylistEntry struct {
	Title           string      `json:"title"`
	Thumbnails      []Thumbnail `json:"thumbnails"`
	Duration        int         `json:"duration"`
	ViewCount       int         `json:"view_count"`
	Timestamp       *int64      `json:"timestamp"`
	LiveStatus      *string     `json:"live_status"`
	Availability    *string     `json:"availability"`
	ChannelURL      string      `json:"channel_url"`
	UploaderURL     string      `json:"uploader_url"`
	Creators        *[]string   `json:"creators"`
	Channel         string      `json:"channel"`
	ChannelID       string      `json:"channel_id"`
	Uploader        string      `json:"uploader"`
	UploaderID      string      `json:"uploader_id"`
	IEKey           string      `json:"ie_key"`
	ID              string      `json:"id"`
	Type            string      `json:"_type"`
	URL             string      `json:"url"`
	XForwardedForIP *string     `json:"__x_forwarded_for_ip"`
}

// Version represents yt-dlp version metadata.
type Version struct {
	Version        string  `json:"version"`
	CurrentGitHead *string `json:"current_git_head"`
	ReleaseGitHead string  `json:"release_git_head"`
	Repository     string  `json:"repository"`
}

// PlaylistMetadata represents the full yt-dlp JSON output for a playlist.
type PlaylistMetadata struct {
	ID                   string            `json:"id"`
	Title                string            `json:"title"`
	Availability         string            `json:"availability"`
	ChannelFollowerCount *int              `json:"channel_follower_count"`
	Description          string            `json:"description"`
	Tags                 []string          `json:"tags"`
	Thumbnails           []Thumbnail       `json:"thumbnails"`
	ModifiedDate         string            `json:"modified_date"`
	ViewCount            int               `json:"view_count"`
	PlaylistCount        int               `json:"playlist_count"`
	Channel              string            `json:"channel"`
	ChannelID            string            `json:"channel_id"`
	UploaderID           string            `json:"uploader_id"`
	Uploader             string            `json:"uploader"`
	ChannelURL           string            `json:"channel_url"`
	UploaderURL          string            `json:"uploader_url"`
	Type                 string            `json:"_type"`
	Entries              []PlaylistEntry   `json:"entries"`
	ExtractorKey         string            `json:"extractor_key"`
	Extractor            string            `json:"extractor"`
	WebpageURL           string            `json:"webpage_url"`
	OriginalURL          string            `json:"original_url"`
	WebpageURLBasename   string            `json:"webpage_url_basename"`
	WebpageURLDomain     string            `json:"webpage_url_domain"`
	ReleaseYear          *int              `json:"release_year"`
	Epoch                int64             `json:"epoch"`
	FilesToMove          map[string]string `json:"__files_to_move"`
	Version              Version           `json:"_version"`
}
