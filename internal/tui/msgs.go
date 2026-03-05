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
	trackID       string
	track         string
	artist        string
	album         string
	shuffleMode   int
	repeatMode    int
	autoplay      bool
	autoplayKnown bool
	playing       bool
	playingKnown  bool
	currentSec    float64
	totalSec      float64
	current       string
	total         string
	progress      float64
	valid         bool
	err           error
}

type playlistsLoadedMsg struct {
	names []string
	ids   map[string]string
	urls  map[string]string
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

type queueTickMsg time.Time

type queueLoadedMsg struct {
	items []upNextRow
	err   error
}

type playbackControlMsg struct {
	action    string
	volume    int
	setVolume bool
	err       error
}

type lyricsLoadedMsg struct {
	text string
	err  error
}

type appStateLoadedMsg struct {
	state persistedState
	err   error
}

type searchSongsLoadedMsg struct {
	query string
	songs []centerSongRow
	err   error
}
