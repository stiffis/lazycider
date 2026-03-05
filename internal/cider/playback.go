package cider

import (
	"encoding/json"
	"fmt"
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
