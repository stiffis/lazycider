package cider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var lrcTimestampRE = regexp.MustCompile(`\[[0-9]{1,2}:[0-9]{2}(?:\.[0-9]{1,3})?\]`)

func (c *Client) Lyrics(trackID, track, artist, album string) (string, string, error) {
	id := strings.TrimSpace(trackID)
	title := strings.TrimSpace(track)
	artistName := strings.TrimSpace(artist)
	albumName := strings.TrimSpace(album)

	failures := make([]string, 0, 3)

	if id != "" {
		text, err := c.ciderLyrics(id)
		if err == nil && strings.TrimSpace(text) != "" {
			return text, "Cider", nil
		}
		if err != nil {
			failures = append(failures, "cider: "+err.Error())
		} else {
			failures = append(failures, "cider: empty lyrics")
		}
	}

	if title != "" && artistName != "" {
		text, err := c.lrclibLyrics(title, artistName, albumName)
		if err == nil && strings.TrimSpace(text) != "" {
			return text, "LRCLIB", nil
		}
		if err != nil {
			failures = append(failures, "lrclib: "+err.Error())
		} else {
			failures = append(failures, "lrclib: empty lyrics")
		}

		text, err = c.lyricsOVHLyrics(title, artistName)
		if err == nil && strings.TrimSpace(text) != "" {
			return text, "lyrics.ovh", nil
		}
		if err != nil {
			failures = append(failures, "lyrics.ovh: "+err.Error())
		} else {
			failures = append(failures, "lyrics.ovh: empty lyrics")
		}
	}

	if len(failures) == 0 {
		if id == "" {
			return "", "", ErrMissingTrackID
		}
		return "", "", ErrEmptyLyrics
	}
	return "", "", fmt.Errorf("no lyrics found (%s)", strings.Join(failures, "; "))
}

func (c *Client) ciderLyrics(trackID string) (string, error) {
	body, _, err := c.doGET("/api/v1/lyrics/" + url.PathEscape(trackID))
	if err != nil {
		return "", err
	}
	text := parseLyricsText(body)
	if strings.TrimSpace(text) == "" {
		return "", ErrEmptyLyrics
	}
	return text, nil
}

func (c *Client) lrclibLyrics(track, artist, album string) (string, error) {
	q := url.Values{}
	q.Set("track_name", track)
	q.Set("artist_name", artist)
	if album != "" {
		q.Set("album_name", album)
	}
	body, _, err := c.externalGET("https://lrclib.net/api/get?" + q.Encode())
	if err != nil {
		return "", err
	}
	text := parseLRCLIBText(body)
	if strings.TrimSpace(text) == "" {
		return "", ErrEmptyLyrics
	}
	return text, nil
}

func (c *Client) lyricsOVHLyrics(track, artist string) (string, error) {
	path := "https://api.lyrics.ovh/v1/" + url.PathEscape(artist) + "/" + url.PathEscape(track)
	body, _, err := c.externalGET(path)
	if err != nil {
		return "", err
	}
	text := parseLyricsOVHText(body)
	if strings.TrimSpace(text) == "" {
		return "", ErrEmptyLyrics
	}
	return text, nil
}

func (c *Client) externalGET(fullURL string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "lazycider/0.1 (+https://github.com/ciderapp)")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 300 {
		return body, resp.StatusCode, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, resp.StatusCode, nil
}

func parseLRCLIBText(body []byte) string {
	var payload struct {
		SyncedLyrics string `json:"syncedLyrics"`
		PlainLyrics  string `json:"plainLyrics"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if strings.TrimSpace(payload.SyncedLyrics) != "" {
		return stripLRCTimestamps(payload.SyncedLyrics)
	}
	return stripLRCTimestamps(payload.PlainLyrics)
}

func parseLyricsOVHText(body []byte) string {
	var payload struct {
		Lyrics string `json:"lyrics"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return stripLRCTimestamps(payload.Lyrics)
}

func stripLRCTimestamps(text string) string {
	rawLines := strings.Split(strings.TrimSpace(text), "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		clean := strings.TrimSpace(lrcTimestampRE.ReplaceAllString(line, ""))
		lines = append(lines, clean)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func parseLyricsText(body []byte) string {
	raw := strings.TrimSpace(string(body))
	if raw == "" || raw == "[]" || raw == "{}" {
		return ""
	}

	var anyJSON any
	if err := json.Unmarshal(body, &anyJSON); err != nil {
		return raw
	}

	lines := make([]string, 0, 64)
	collectLyricStrings(anyJSON, &lines)
	if len(lines) == 0 {
		return ""
	}

	seen := make(map[string]bool, len(lines))
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		out = append(out, line)
	}

	return strings.Join(out, "\n")
}

func collectLyricStrings(node any, out *[]string) {
	switch v := node.(type) {
	case map[string]any:
		for k, val := range v {
			lk := strings.ToLower(k)
			switch vv := val.(type) {
			case string:
				if lk == "text" || lk == "line" || lk == "lyric" || lk == "lyrics" || lk == "content" {
					*out = append(*out, vv)
				}
			default:
				collectLyricStrings(vv, out)
			}
		}
	case []any:
		for _, item := range v {
			collectLyricStrings(item, out)
		}
	case string:
		if strings.Contains(v, "\n") {
			*out = append(*out, v)
		}
	}
}
