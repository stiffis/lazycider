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

		totalSec := float64(np.DurationMS) / 1000.0
		if totalSec <= 0 {
			totalSec = np.CurrentSec + np.RemainingSec
		}
		if totalSec <= 0 {
			return playbackLoadedMsg{
				trackID: strings.TrimSpace(np.TrackID),
				track:   strings.TrimSpace(np.Track),
				artist:  strings.TrimSpace(np.Artist),
				album:   strings.TrimSpace(np.Album),
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
			trackID:  strings.TrimSpace(np.TrackID),
			track:    strings.TrimSpace(np.Track),
			artist:   strings.TrimSpace(np.Artist),
			album:    strings.TrimSpace(np.Album),
			current:  formatClock(currentSec),
			total:    formatClock(totalSec),
			progress: progress,
			valid:    true,
		}
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

func fetchLyricsCmd(client *cider.Client, trackID, track, artist string) tea.Cmd {
	return func() tea.Msg {
		text, err := client.Lyrics(trackID)
		if err != nil {
			reason := fmt.Sprintf("lyrics endpoint unavailable (%v)", err)
			return lyricsLoadedMsg{text: fallbackLyrics(track, artist, reason)}
		}
		if strings.TrimSpace(text) == "" {
			return lyricsLoadedMsg{text: fallbackLyrics(track, artist, "lyrics endpoint returned empty response")}
		}
		return lyricsLoadedMsg{text: text}
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

func playTrackInContextCmd(client *cider.Client, trackID, playlistURL string, enqueueIDs []string) tea.Cmd {
	id := strings.TrimSpace(trackID)
	contextURL := strings.TrimSpace(playlistURL)
	return func() tea.Msg {
		if id == "" {
			return playItemResultMsg{trackID: id, err: fmt.Errorf("missing track id")}
		}
		if contextURL != "" {
			if u := cider.PlaylistTrackURL(contextURL, id); strings.TrimSpace(u) != "" {
				if err := client.PlayURL(u); err == nil {
					return playItemResultMsg{trackID: id}
				}
			}
		}
		if len(enqueueIDs) > 0 {
			_ = client.ClearQueue()
		}
		err := client.PlayItem("songs", id)
		if err == nil {
			for _, qid := range enqueueIDs {
				if strings.TrimSpace(qid) == "" || strings.TrimSpace(qid) == id {
					continue
				}
				if qErr := client.PlayLater("songs", qid); qErr != nil {
					break
				}
			}
		}
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
		if currentIdx >= 0 && currentIdx < len(tracks) {
			start = currentIdx + 1
		} else if nowID != "" {
			for i, t := range tracks {
				if strings.TrimSpace(t.ID) == nowID {
					start = i + 1
					break
				}
			}
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
		"API lyrics are not available right now.",
		"",
		"This panel is ready for full synced lyrics",
		"as soon as Cider returns non-empty data.",
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
	if m.rightPanelMode == RightPanelLyrics {
		return nil
	}

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
