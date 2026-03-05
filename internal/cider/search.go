package cider

import (
	"net/url"
	"os"
	"strconv"
	"strings"

	"lazycider/internal/music"
)

func (c *Client) SearchSongs(query string, limit int) ([]music.Track, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	tracks := make([]music.Track, 0, limit*2)
	seen := make(map[string]struct{}, limit*2)

	libraryPath := "/v1/me/library/search?term=" + url.QueryEscape(q) + "&types=library-songs&limit=" + url.QueryEscape(strconv.Itoa(limit))
	if root, err := c.runV3(libraryPath); err == nil {
		tracks = appendSearchSongBucket(tracks, seen, root, "library-songs")
	}

	storefront := strings.TrimSpace(os.Getenv("CIDER_STOREFRONT"))
	if storefront == "" {
		storefront = "us"
	}
	catalogPath := "/v1/catalog/" + url.PathEscape(storefront) + "/search?term=" + url.QueryEscape(q) + "&types=songs&limit=" + url.QueryEscape(strconv.Itoa(limit))
	root, err := c.runV3(catalogPath)
	if err != nil {
		if len(tracks) > 0 {
			return tracks, nil
		}
		return nil, err
	}
	tracks = appendSearchSongBucket(tracks, seen, root, "songs")
	return tracks, nil
}

func appendSearchSongBucket(out []music.Track, seen map[string]struct{}, root map[string]any, key string) []music.Track {
	results, _ := extractMapAny(root, "data", "results").(map[string]any)
	if results == nil {
		return out
	}
	bucket, _ := results[key].(map[string]any)
	if bucket == nil {
		return out
	}
	rows, _ := bucket["data"].([]any)
	for _, row := range rows {
		it, _ := row.(map[string]any)
		if it == nil {
			continue
		}
		id := strings.TrimSpace(extractMapString(it, "id"))
		title := strings.TrimSpace(extractMapString(it, "attributes", "name"))
		if id == "" || title == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		artist := strings.TrimSpace(extractMapString(it, "attributes", "artistName"))
		album := strings.TrimSpace(extractMapString(it, "attributes", "albumName"))
		urlStr := strings.TrimSpace(extractMapString(it, "attributes", "url"))
		dur, _ := anyToInt64(extractMapAny(it, "attributes", "durationInMillis"))
		out = append(out, music.Track{ID: id, Title: title, Artist: artist, Album: album, URL: urlStr, DurationMS: dur})
	}
	return out
}
