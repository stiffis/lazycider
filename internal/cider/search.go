package cider

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"lazycider/internal/music"
)

func (c *Client) SearchAll(query string, limit int) (map[string][]music.SearchResult, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return map[string][]music.SearchResult{"songs": nil, "artists": nil, "albums": nil, "playlists": nil}, nil
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	out := map[string][]music.SearchResult{
		"songs":     make([]music.SearchResult, 0, limit*2),
		"artists":   make([]music.SearchResult, 0, limit),
		"albums":    make([]music.SearchResult, 0, limit),
		"playlists": make([]music.SearchResult, 0, limit),
	}

	seen := map[string]map[string]struct{}{
		"songs":     make(map[string]struct{}, limit*2),
		"artists":   make(map[string]struct{}, limit),
		"albums":    make(map[string]struct{}, limit),
		"playlists": make(map[string]struct{}, limit),
	}

	libraryPath := "/v1/me/library/search?term=" + url.QueryEscape(q) + "&types=library-songs&limit=" + url.QueryEscape(strconv.Itoa(limit))
	if root, err := c.runV3(libraryPath); err == nil {
		out["songs"] = appendSearchBucket(out["songs"], seen["songs"], root, "library-songs", "library-songs")
	}

	storefront := ciderStorefront()
	catalogPath := "/v1/catalog/" + url.PathEscape(storefront) + "/search?term=" + url.QueryEscape(q) + "&types=songs,artists,albums,playlists&limit=" + url.QueryEscape(strconv.Itoa(limit))
	root, err := c.runV3(catalogPath)
	if err != nil {
		if len(out["songs"]) > 0 {
			return out, nil
		}
		return nil, err
	}

	out["songs"] = appendSearchBucket(out["songs"], seen["songs"], root, "songs", "songs")
	out["artists"] = appendSearchBucket(out["artists"], seen["artists"], root, "artists", "artists")
	out["albums"] = appendSearchBucket(out["albums"], seen["albums"], root, "albums", "albums")
	out["playlists"] = appendSearchBucket(out["playlists"], seen["playlists"], root, "playlists", "playlists")

	return out, nil
}

func (c *Client) SearchDetail(kind, id string) (music.SearchDetail, error) {
	k := strings.TrimSpace(kind)
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return music.SearchDetail{}, fmt.Errorf("missing id")
	}

	sf := ciderStorefront()
	switch k {
	case "artists", "library-artists":
		return c.fetchArtistDetail(sf, cleanID)
	case "albums", "library-albums":
		return c.fetchAlbumDetail(sf, cleanID)
	case "playlists", "library-playlists":
		return c.fetchPlaylistDetail(sf, cleanID)
	default:
		return music.SearchDetail{}, fmt.Errorf("detail not supported for type %s", k)
	}
}

func (c *Client) SearchSongs(query string, limit int) ([]music.Track, error) {
	buckets, err := c.SearchAll(query, limit)
	if err != nil {
		return nil, err
	}
	results := buckets["songs"]
	out := make([]music.Track, 0, len(results))
	for _, r := range results {
		out = append(out, music.Track{ID: r.ID, Title: r.Title, Artist: r.Artist, Album: r.Album, URL: r.URL, DurationMS: r.DurationMS})
	}
	return out, nil
}

func appendSearchBucket(out []music.SearchResult, seen map[string]struct{}, root map[string]any, key, typ string) []music.SearchResult {
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
		out = append(out, music.SearchResult{ID: id, Type: typ, Title: title, Artist: artist, Album: album, URL: urlStr, DurationMS: dur})
	}
	return out
}

func (c *Client) fetchAlbumDetail(storefront, id string) (music.SearchDetail, error) {
	metaPath := "/v1/catalog/" + url.PathEscape(storefront) + "/albums/" + url.PathEscape(id)
	metaRoot, metaErr := c.runV3(metaPath)
	detail := music.SearchDetail{Type: "albums"}
	if metaErr == nil {
		items, _ := parseRunV3DataAndNext(metaRoot)
		if len(items) > 0 {
			it := items[0]
			detail.Title = strings.TrimSpace(extractMapString(it, "attributes", "name"))
			detail.Subtitle = strings.TrimSpace(extractMapString(it, "attributes", "artistName"))
			detail.Description = strings.TrimSpace(extractMapString(it, "attributes", "url"))
		}
	}
	tracks, err := c.fetchTracksPage("/v1/catalog/" + url.PathEscape(storefront) + "/albums/" + url.PathEscape(id) + "/tracks?limit=100")
	if err != nil {
		if detail.Title != "" {
			return detail, nil
		}
		return music.SearchDetail{}, err
	}
	detail.Tracks = tracks
	if detail.Title == "" {
		detail.Title = "Album"
	}
	return detail, nil
}

func (c *Client) fetchPlaylistDetail(storefront, id string) (music.SearchDetail, error) {
	metaPath := "/v1/catalog/" + url.PathEscape(storefront) + "/playlists/" + url.PathEscape(id)
	metaRoot, metaErr := c.runV3(metaPath)
	detail := music.SearchDetail{Type: "playlists"}
	if metaErr == nil {
		items, _ := parseRunV3DataAndNext(metaRoot)
		if len(items) > 0 {
			it := items[0]
			detail.Title = strings.TrimSpace(extractMapString(it, "attributes", "name"))
			detail.Subtitle = strings.TrimSpace(extractMapString(it, "attributes", "curatorName"))
			detail.Description = strings.TrimSpace(extractMapString(it, "attributes", "url"))
		}
	}
	tracks, err := c.fetchTracksPage("/v1/catalog/" + url.PathEscape(storefront) + "/playlists/" + url.PathEscape(id) + "/tracks?limit=100")
	if err != nil {
		if detail.Title != "" {
			return detail, nil
		}
		return music.SearchDetail{}, err
	}
	detail.Tracks = tracks
	if detail.Title == "" {
		detail.Title = "Playlist"
	}
	return detail, nil
}

func (c *Client) fetchArtistDetail(storefront, id string) (music.SearchDetail, error) {
	metaPath := "/v1/catalog/" + url.PathEscape(storefront) + "/artists/" + url.PathEscape(id)
	metaRoot, err := c.runV3(metaPath)
	if err != nil {
		return music.SearchDetail{}, err
	}
	items, _ := parseRunV3DataAndNext(metaRoot)
	if len(items) == 0 {
		return music.SearchDetail{}, fmt.Errorf("artist not found")
	}
	it := items[0]
	name := strings.TrimSpace(extractMapString(it, "attributes", "name"))
	genre := ""
	if genres, ok := extractMapAny(it, "attributes", "genreNames").([]any); ok && len(genres) > 0 {
		if g0, ok := genres[0].(string); ok {
			genre = strings.TrimSpace(g0)
		}
	}
	urlStr := strings.TrimSpace(extractMapString(it, "attributes", "url"))
	detail := music.SearchDetail{Type: "artists", Title: name, Subtitle: genre, Description: urlStr}

	top, topErr := c.fetchTracksPage("/v1/catalog/" + url.PathEscape(storefront) + "/artists/" + url.PathEscape(id) + "/view/top-songs?limit=25")
	if topErr == nil {
		detail.Tracks = top
	}
	return detail, nil
}

func (c *Client) fetchTracksPage(path string) ([]music.Track, error) {
	root, err := c.runV3(path)
	if err != nil {
		return nil, err
	}
	items, _ := parseRunV3DataAndNext(root)
	out := make([]music.Track, 0, len(items))
	for _, it := range items {
		title := strings.TrimSpace(extractMapString(it, "attributes", "name"))
		if title == "" {
			continue
		}
		artist := strings.TrimSpace(extractMapString(it, "attributes", "artistName"))
		album := strings.TrimSpace(extractMapString(it, "attributes", "albumName"))
		urlStr := strings.TrimSpace(extractMapString(it, "attributes", "url"))
		dur, _ := anyToInt64(extractMapAny(it, "attributes", "durationInMillis"))
		id := strings.TrimSpace(extractMapString(it, "id"))
		out = append(out, music.Track{ID: id, Title: title, Artist: artist, Album: album, URL: urlStr, DurationMS: dur})
	}
	return out, nil
}

func ciderStorefront() string {
	storefront := strings.TrimSpace(os.Getenv("CIDER_STOREFRONT"))
	if storefront == "" {
		storefront = "us"
	}
	return storefront
}
