package cider

import (
	"fmt"
	"strings"

	"lazycider/internal/music"
)

func (c *Client) ListLibraryPlaylists() ([]music.Playlist, error) {
	path := "/v1/me/library/playlists?limit=100"
	out := make([]music.Playlist, 0, 64)

	for pages := 0; pages < 20; pages++ {
		root, err := c.runV3(path)
		if err != nil {
			return nil, err
		}
		items, next := parseRunV3DataAndNext(root)
		for _, it := range items {
			name := strings.TrimSpace(extractMapString(it, "attributes", "name"))
			id := strings.TrimSpace(extractMapString(it, "id"))
			if name == "" {
				continue
			}
			out = append(out, music.Playlist{ID: id, Name: name})
		}
		if strings.TrimSpace(next) == "" {
			break
		}
		path = next
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no playlists found")
	}
	return out, nil
}

func (c *Client) ListPlaylistTracks(playlistID string) ([]music.Track, error) {
	id := strings.TrimSpace(playlistID)
	if id == "" {
		return nil, fmt.Errorf("missing playlist id")
	}

	path := "/v1/me/library/playlists/" + id + "/tracks?limit=100"
	tracks := make([]music.Track, 0, 128)

	for pages := 0; pages < 20; pages++ {
		root, err := c.runV3(path)
		if err != nil {
			return nil, err
		}
		items, next := parseRunV3DataAndNext(root)
		for _, it := range items {
			title := strings.TrimSpace(extractMapString(it, "attributes", "name"))
			if title == "" {
				continue
			}
			artist := strings.TrimSpace(extractMapString(it, "attributes", "artistName"))
			album := strings.TrimSpace(extractMapString(it, "attributes", "albumName"))
			dur, _ := anyToInt64(extractMapAny(it, "attributes", "durationInMillis"))
			trackID := strings.TrimSpace(extractMapString(it, "id"))
			tracks = append(tracks, music.Track{
				ID:         trackID,
				Title:      title,
				Artist:     artist,
				Album:      album,
				DurationMS: dur,
			})
		}
		if strings.TrimSpace(next) == "" {
			break
		}
		path = next
	}

	return tracks, nil
}
