package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"lazycider/internal/artwork"
	"lazycider/internal/cider"
	"lazycider/internal/music"
	"lazycider/internal/term/kitty"
)

const (
	coverPollInterval    = 6 * time.Second
	playbackPollInterval = 1 * time.Second
	queuePollInterval    = 3 * time.Second
)

func coverTickCmd() tea.Cmd {
	return tea.Tick(coverPollInterval, func(t time.Time) tea.Msg {
		return coverTickMsg(t)
	})
}

func playbackTickCmd() tea.Cmd {
	return tea.Tick(playbackPollInterval, func(t time.Time) tea.Msg {
		return playbackTickMsg(t)
	})
}

func queueTickCmd() tea.Cmd {
	return tea.Tick(queuePollInterval, func(t time.Time) tea.Msg {
		return queueTickMsg(t)
	})
}

func fetchCoverCmd(client *cider.Client) tea.Cmd {
	return func() tea.Msg {
		np, err := client.NowPlaying()
		if err != nil {
			return coverLoadedMsg{err: err}
		}

		if strings.TrimSpace(np.ArtworkURL) == "" {
			fallback, qErr := client.QueueHeadWithArtwork()
			if qErr != nil {
				return coverLoadedMsg{err: qErr}
			}
			np.ArtworkURL = fallback.ArtworkURL
			if np.TrackID == "" {
				np.TrackID = fallback.TrackID
			}
			if np.Track == "" {
				np.Track = fallback.Track
			}
			if np.Artist == "" {
				np.Artist = fallback.Artist
			}
			if np.Album == "" {
				np.Album = fallback.Album
			}
		}

		path, w, h, err := artwork.Cache(np.ArtworkURL)
		if err != nil {
			return coverLoadedMsg{err: err}
		}

		return coverLoadedMsg{
			coverURL:  np.ArtworkURL,
			coverPath: path,
			coverW:    w,
			coverH:    h,
			trackID:   np.TrackID,
			track:     np.Track,
			artist:    np.Artist,
			album:     np.Album,
		}
	}
}

func fetchPlaybackCmd(client *cider.Client) tea.Cmd {
	return func() tea.Msg {
		np, err := client.NowPlaying()
		if err != nil {
			return playbackLoadedMsg{err: err}
		}
		autoplay, aErr := client.Autoplay()
		isPlaying, pErr := client.IsPlaying()

		totalSec := float64(np.DurationMS) / 1000.0
		if totalSec <= 0 {
			totalSec = np.CurrentSec + np.RemainingSec
		}
		if totalSec <= 0 {
			return playbackLoadedMsg{
				trackID:       strings.TrimSpace(np.TrackID),
				track:         strings.TrimSpace(np.Track),
				artist:        strings.TrimSpace(np.Artist),
				album:         strings.TrimSpace(np.Album),
				shuffleMode:   np.ShuffleMode,
				repeatMode:    np.RepeatMode,
				autoplay:      autoplay,
				autoplayKnown: aErr == nil,
				playing:       isPlaying,
				playingKnown:  pErr == nil,
			}
		}

		currentSec := np.CurrentSec
		if currentSec < 0 {
			currentSec = 0
		}
		if currentSec > totalSec {
			currentSec = totalSec
		}

		progress := currentSec / totalSec
		if progress < 0 {
			progress = 0
		}
		if progress > 1 {
			progress = 1
		}

		return playbackLoadedMsg{
			trackID:       strings.TrimSpace(np.TrackID),
			track:         strings.TrimSpace(np.Track),
			artist:        strings.TrimSpace(np.Artist),
			album:         strings.TrimSpace(np.Album),
			shuffleMode:   np.ShuffleMode,
			repeatMode:    np.RepeatMode,
			autoplay:      autoplay,
			autoplayKnown: aErr == nil,
			playing:       isPlaying,
			playingKnown:  pErr == nil,
			currentSec:    currentSec,
			totalSec:      totalSec,
			current:       formatClock(currentSec),
			total:         formatClock(totalSec),
			progress:      progress,
			valid:         true,
		}
	}
}

func playPauseCmd(client *cider.Client) tea.Cmd {
	return func() tea.Msg {
		err := client.PlayPause()
		return playbackControlMsg{action: "playpause", err: err}
	}
}

func fetchPlaylistsCmd(client *cider.Client) tea.Cmd {
	return func() tea.Msg {
		playlists, err := client.ListLibraryPlaylists()
		if err != nil {
			return playlistsLoadedMsg{err: err}
		}
		names := make([]string, 0, len(playlists))
		ids := make(map[string]string, len(playlists))
		urls := make(map[string]string, len(playlists))
		for _, p := range playlists {
			names = append(names, p.Name)
			if strings.TrimSpace(p.ID) != "" {
				ids[p.Name] = p.ID
			}
			if strings.TrimSpace(p.URL) != "" {
				urls[p.Name] = p.URL
			}
		}
		return playlistsLoadedMsg{names: names, ids: ids, urls: urls}
	}
}

func fetchPlaylistTracksCmd(client *cider.Client, name, playlistID string) tea.Cmd {
	return func() tea.Msg {
		id := strings.TrimSpace(playlistID)
		if id == "" {
			return playlistTracksLoadedMsg{name: name, err: fmt.Errorf("missing playlist id for %q", name)}
		}

		tracks, err := client.ListPlaylistTracks(id)
		if err != nil {
			return playlistTracksLoadedMsg{name: name, err: err}
		}

		songs := make([]centerSongRow, 0, len(tracks))
		for _, t := range tracks {
			artist := strings.TrimSpace(t.Artist)
			if artist == "" {
				artist = strings.TrimSpace(t.Album)
			}
			songs = append(songs, centerSongRow{
				ID:       strings.TrimSpace(t.ID),
				URL:      strings.TrimSpace(t.URL),
				Title:    t.Title,
				Artist:   artist,
				Duration: formatTrackDuration(t),
			})
		}
		if len(songs) == 0 {
			songs = []centerSongRow{{Title: "Empty playlist", Artist: name, Duration: ""}}
		}

		return playlistTracksLoadedMsg{name: name, songs: songs}
	}
}

func fetchLyricsCmd(client *cider.Client, trackID, track, artist, album string) tea.Cmd {
	return func() tea.Msg {
		text, source, err := client.Lyrics(trackID, track, artist, album)
		if err != nil {
			reason := fmt.Sprintf("cider/lrclib/lyrics.ovh unavailable (%v)", err)
			return lyricsLoadedMsg{text: fallbackLyrics(track, artist, reason)}
		}
		if strings.TrimSpace(text) == "" {
			return lyricsLoadedMsg{text: fallbackLyrics(track, artist, "all lyrics providers returned empty response")}
		}
		if strings.TrimSpace(source) != "" {
			text = "[Source: " + strings.TrimSpace(source) + "]\n\n" + text
		}
		return lyricsLoadedMsg{text: text}
	}
}

func searchSongsCmd(client *cider.Client, query string) tea.Cmd {
	q := strings.TrimSpace(query)
	return func() tea.Msg {
		if q == "" {
			return searchSongsLoadedMsg{query: q, songs: nil}
		}

		tracks, err := client.SearchSongs(q, 25)
		if err != nil {
			return searchSongsLoadedMsg{query: q, err: err}
		}

		rows := make([]centerSongRow, 0, len(tracks))
		for _, t := range tracks {
			artist := strings.TrimSpace(t.Artist)
			if artist == "" {
				artist = strings.TrimSpace(t.Album)
			}
			rows = append(rows, centerSongRow{
				ID:       strings.TrimSpace(t.ID),
				URL:      strings.TrimSpace(t.URL),
				Title:    strings.TrimSpace(t.Title),
				Artist:   artist,
				Duration: formatTrackDuration(t),
			})
		}
		return searchSongsLoadedMsg{query: q, songs: rows}
	}
}

func playItemCmd(client *cider.Client, trackID string) tea.Cmd {
	id := strings.TrimSpace(trackID)
	return func() tea.Msg {
		if id == "" {
			return playItemResultMsg{trackID: id, err: fmt.Errorf("missing track id")}
		}
		err := client.PlayItem("songs", id)
		return playItemResultMsg{trackID: id, err: err}
	}
}

func playTrackCmd(client *cider.Client, trackID string) tea.Cmd {
	id := strings.TrimSpace(trackID)
	return func() tea.Msg {
		if id == "" {
			return playItemResultMsg{trackID: id, err: fmt.Errorf("missing track id")}
		}
		err := client.PlayItem("songs", id)
		return playItemResultMsg{trackID: id, err: err}
	}
}

func fetchQueueCmd(client *cider.Client, nowPlayingTrackID string) tea.Cmd {
	nowID := strings.TrimSpace(nowPlayingTrackID)
	return func() tea.Msg {
		tracks, currentIdx, err := client.QueueTracks()
		if err != nil {
			return queueLoadedMsg{err: err}
		}
		if len(tracks) == 0 {
			return queueLoadedMsg{items: nil}
		}

		start := 0
		foundCurrent := false
		if currentIdx >= 0 && currentIdx < len(tracks) {
			start = currentIdx + 1
			foundCurrent = true
		} else if nowID != "" {
			for i, t := range tracks {
				if strings.TrimSpace(t.ID) == nowID {
					start = i + 1
					foundCurrent = true
					break
				}
			}
		}
		if !foundCurrent {
			return queueLoadedMsg{items: nil}
		}
		if start < 0 {
			start = 0
		}
		if start > len(tracks) {
			start = len(tracks)
		}

		rows := make([]upNextRow, 0, len(tracks)-start)
		for _, t := range tracks[start:] {
			sub := strings.TrimSpace(t.Artist)
			if sub == "" {
				sub = strings.TrimSpace(t.Album)
			}
			if sub == "" {
				sub = "Unknown artist"
			}
			rows = append(rows, upNextRow{Title: t.Title, Subtitle: sub})
		}
		return queueLoadedMsg{items: rows}
	}
}

func toggleShuffleCmd(client *cider.Client) tea.Cmd {
	return func() tea.Msg {
		err := client.ToggleShuffle()
		return playbackControlMsg{action: "shuffle", err: err}
	}
}

func toggleRepeatCmd(client *cider.Client) tea.Cmd {
	return func() tea.Msg {
		err := client.ToggleRepeat()
		return playbackControlMsg{action: "repeat", err: err}
	}
}

func toggleAutoplayCmd(client *cider.Client) tea.Cmd {
	return func() tea.Msg {
		err := client.ToggleAutoplay()
		return playbackControlMsg{action: "autoplay", err: err}
	}
}

func nextCmd(client *cider.Client) tea.Cmd {
	return func() tea.Msg {
		err := client.Next()
		return playbackControlMsg{action: "next", err: err}
	}
}

func previousCmd(client *cider.Client) tea.Cmd {
	return func() tea.Msg {
		err := client.Previous()
		return playbackControlMsg{action: "previous", err: err}
	}
}

func adjustVolumeCmd(client *cider.Client, current, delta int) tea.Cmd {
	target := current + delta
	if target < 0 {
		target = 0
	}
	if target > 100 {
		target = 100
	}
	return func() tea.Msg {
		err := client.SetVolumePercent(target)
		return playbackControlMsg{action: "volume", volume: target, setVolume: true, err: err}
	}
}

func fallbackLyrics(track, artist, reason string) string {
	title := strings.TrimSpace(track)
	if title == "" {
		title = "Current Track"
	}
	by := strings.TrimSpace(artist)
	if by == "" {
		by = "Unknown Artist"
	}

	return strings.Join([]string{
		title + " — " + by,
		"",
		"[Lyrics fallback]",
		"No provider returned lyrics right now.",
		"",
		"Provider order:",
		"1) Cider local API",
		"2) LRCLIB",
		"3) lyrics.ovh",
		"",
		"Reason:",
		reason,
	}, "\n")
}

func clearKittyImagesCmd() tea.Cmd {
	return func() tea.Msg {
		_ = kitty.Clear()
		return nil
	}
}

func (m Model) drawCoverCmd(clear bool) tea.Cmd {
	if m.coverPath == "" || m.coverW <= 0 || m.coverH <= 0 || m.width <= 0 || m.height <= 0 {
		return nil
	}

	l := m.layoutInfo()
	if l.rightWidth <= 0 || l.rightCoverHeight <= 0 {
		return nil
	}

	drawKey := fmt.Sprintf("%s|%d|%d|%d|%d|%d|%d", m.coverPath, m.coverW, m.coverH, l.rightX, l.rightCoverY, l.rightWidth, l.rightCoverHeight)
	if !clear && drawKey == m.lastCoverKey {
		return nil
	}

	return drawCoverToPanelCmd(m.coverPath, m.coverW, m.coverH, m.width, m.height, l.rightX, l.rightCoverY, l.rightWidth, l.rightCoverHeight, drawKey, clear)
}

func drawCoverToPanelCmd(path string, imgW, imgH, termW, termH, panelX, panelY, panelW, panelH int, drawKey string, clear bool) tea.Cmd {
	return func() tea.Msg {
		err := kitty.Draw(path, kitty.DrawOptions{
			ImageWidth:  imgW,
			ImageHeight: imgH,
			TermWidth:   termW,
			TermHeight:  termH,
			PanelX:      panelX,
			PanelY:      panelY,
			PanelW:      panelW,
			PanelH:      panelH,
			Clear:       clear,
		})
		if err != nil {
			return coverDrawnMsg{drawKey: drawKey, err: err}
		}
		return coverDrawnMsg{drawKey: drawKey}
	}
}

func formatClock(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	total := int(seconds + 0.5)
	min := total / 60
	sec := total % 60
	if sec < 10 {
		return strconv.Itoa(min) + ":0" + strconv.Itoa(sec)
	}
	return strconv.Itoa(min) + ":" + strconv.Itoa(sec)
}

func formatTrackDuration(t music.Track) string {
	if t.DurationMS <= 0 {
		return ""
	}
	sec := t.DurationMS / 1000
	min := sec / 60
	rem := sec % 60
	return fmt.Sprintf("%d:%02d", min, rem)
}
