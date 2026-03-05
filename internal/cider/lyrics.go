package cider

import (
	"encoding/json"
	"net/url"
	"strings"
)

func (c *Client) Lyrics(trackID string) (string, error) {
	id := strings.TrimSpace(trackID)
	if id == "" {
		return "", ErrMissingTrackID
	}
	body, _, err := c.doGET("/api/v1/lyrics/" + url.PathEscape(id))
	if err != nil {
		return "", err
	}
	text := parseLyricsText(body)
	if strings.TrimSpace(text) == "" {
		return "", ErrEmptyLyrics
	}
	return text, nil
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
