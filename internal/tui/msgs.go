package tui

import "time"

type coverTickMsg time.Time

type coverLoadedMsg struct {
	coverURL  string
	coverPath string
	coverW    int
	coverH    int
	trackID   string
	track     string
	artist    string
	album     string
	err       error
}

type coverDrawnMsg struct {
	drawKey string
	err     error
}

type playbackTickMsg time.Time

type playbackLoadedMsg struct {
	trackID  string
	track    string
	artist   string
	album    string
	current  string
	total    string
	progress float64
	valid    bool
	err      error
}

type playlistsLoadedMsg struct {
	names []string
	ids   map[string]string
	err   error
}

type playlistTracksLoadedMsg struct {
	name  string
	songs []centerSongRow
	err   error
}

type playItemResultMsg struct {
	trackID string
	err     error
}

type lyricsLoadedMsg struct {
	text string
	err  error
}
