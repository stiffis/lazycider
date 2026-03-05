package cider

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"lazycider/internal/music"
)

type nowPlayingResponse struct {
	Info struct {
		Name                string  `json:"name"`
		Artist              string  `json:"artistName"`
		Album               string  `json:"albumName"`
		DurationInMillis    int64   `json:"durationInMillis"`
		CurrentPlaybackTime float64 `json:"currentPlaybackTime"`
		RemainingTime       float64 `json:"remainingTime"`
		Play                struct {
			ID string `json:"id"`
		} `json:"playParams"`
		Artwork struct {
			URL string `json:"url"`
		} `json:"artwork"`
	} `json:"info"`
}

func (c *Client) NowPlaying() (music.NowPlaying, error) {
	body, _, err := c.doGET("/api/v1/playback/now-playing")
	if err != nil {
		return music.NowPlaying{}, err
	}

	var resp nowPlayingResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return music.NowPlaying{}, err
	}

	np := music.NowPlaying{
		TrackID:      strings.TrimSpace(resp.Info.Play.ID),
		Track:        strings.TrimSpace(resp.Info.Name),
		Artist:       strings.TrimSpace(resp.Info.Artist),
		Album:        strings.TrimSpace(resp.Info.Album),
		ArtworkURL:   normalizeArtworkURL(resp.Info.Artwork.URL),
		DurationMS:   resp.Info.DurationInMillis,
		CurrentSec:   resp.Info.CurrentPlaybackTime,
		RemainingSec: resp.Info.RemainingTime,
	}
	return np, nil
}

func (c *Client) QueueHeadWithArtwork() (music.NowPlaying, error) {
	body, _, err := c.doGET("/api/v1/playback/queue")
	if err != nil {
		return music.NowPlaying{}, err
	}

	var raw []map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return music.NowPlaying{}, err
	}

	for _, item := range raw {
		art := normalizeArtworkURL(extractArtworkURL(item))
		if art == "" {
			continue
		}
		trackID := strings.TrimSpace(extractMapString(item, "attributes", "playParams", "id"))
		if trackID == "" {
			trackID = strings.TrimSpace(extractMapString(item, "id"))
		}
		track := strings.TrimSpace(extractMapString(item, "attributes", "name"))
		artist := strings.TrimSpace(extractMapString(item, "attributes", "artistName"))
		album := strings.TrimSpace(extractMapString(item, "attributes", "albumName"))
		return music.NowPlaying{
			TrackID:    trackID,
			Track:      track,
			Artist:     artist,
			Album:      album,
			ArtworkURL: art,
		}, nil
	}

	return music.NowPlaying{}, fmt.Errorf("queue missing usable artwork")
}

func (c *Client) PlayItem(itemType, id string) error {
	t := strings.TrimSpace(itemType)
	if t == "" {
		t = "songs"
	}
	payload := map[string]string{
		"type": t,
		"id":   strings.TrimSpace(id),
	}
	if payload["id"] == "" {
		return fmt.Errorf("missing item id")
	}
	_, _, err := c.doPOSTJSON("/api/v1/playback/play-item", payload)
	return err
}

func (c *Client) PlayURL(rawURL string) error {
	u := strings.TrimSpace(rawURL)
	if u == "" {
		return fmt.Errorf("missing url")
	}
	_, _, err := c.doPOSTJSON("/api/v1/playback/play-url", map[string]string{"url": u})
	return err
}

func (c *Client) PlayLater(itemType, id string) error {
	t := strings.TrimSpace(itemType)
	if t == "" {
		t = "songs"
	}
	payload := map[string]string{
		"type": t,
		"id":   strings.TrimSpace(id),
	}
	if payload["id"] == "" {
		return fmt.Errorf("missing item id")
	}
	_, _, err := c.doPOSTJSON("/api/v1/playback/play-later", payload)
	return err
}

func (c *Client) ClearQueue() error {
	_, _, err := c.doPOSTJSON("/api/v1/playback/queue/clear-queue", map[string]any{})
	return err
}

func (c *Client) QueueTracks() ([]music.Track, int, error) {
	body, _, err := c.doGET("/api/v1/playback/queue")
	if err != nil {
		return nil, -1, err
	}

	var raw []map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, -1, err
	}

	out := make([]music.Track, 0, len(raw))
	currentIndex := -1
	for _, item := range raw {
		title := strings.TrimSpace(extractMapString(item, "attributes", "name"))
		if title == "" {
			continue
		}
		artist := strings.TrimSpace(extractMapString(item, "attributes", "artistName"))
		album := strings.TrimSpace(extractMapString(item, "attributes", "albumName"))
		id := strings.TrimSpace(extractMapString(item, "attributes", "playParams", "id"))
		if id == "" {
			id = strings.TrimSpace(extractMapString(item, "id"))
		}
		dur, _ := anyToInt64(extractMapAny(item, "attributes", "durationInMillis"))
		url := strings.TrimSpace(extractMapString(item, "attributes", "url"))

		isCurrent := false
		if st, ok := extractMapAny(item, "_state").(map[string]any); ok {
			if v, ok := anyToInt64(st["current"]); ok && v > 0 {
				isCurrent = true
			}
		}
		out = append(out, music.Track{ID: id, Title: title, Artist: artist, Album: album, DurationMS: dur, URL: url})
		if isCurrent && currentIndex < 0 {
			currentIndex = len(out) - 1
		}
	}

	return out, currentIndex, nil
}

func PlaylistTrackURL(playlistURL, trackID string) string {
	base := strings.TrimSpace(playlistURL)
	id := strings.TrimSpace(trackID)
	if base == "" || id == "" {
		return ""
	}
	u, err := url.Parse(base)
	if err != nil {
		return ""
	}
	q := u.Query()
	q.Set("i", id)
	u.RawQuery = q.Encode()
	return u.String()
}

func extractArtworkURL(m map[string]any) string {
	if v := extractMapString(m, "attributes", "artwork", "url"); v != "" {
		return v
	}
	if v := extractMapString(m, "artwork", "url"); v != "" {
		return v
	}
	if v := extractMapString(m, "assets", "0", "artworkURL"); v != "" {
		return v
	}
	return ""
}
