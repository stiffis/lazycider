package cider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewFromEnv() *Client {
	base := strings.TrimRight(os.Getenv("CIDER_API_BASE"), "/")
	if base == "" {
		base = "http://localhost:10767"
	}
	return New(base, strings.TrimSpace(os.Getenv("CIDER_API_TOKEN")))
}

func New(baseURL, token string) *Client {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = "http://localhost:10767"
	}
	return &Client{
		baseURL: base,
		token:   strings.TrimSpace(token),
		http:    &http.Client{Timeout: 8 * time.Second},
	}
}

func (c *Client) doGET(path string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, 0, err
	}
	if c.token != "" {
		req.Header.Set("apitoken", c.token)
	}
	req.Header.Set("Accept", "application/json")

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

func (c *Client) runV3(path string) (map[string]any, error) {
	payload := map[string]string{"path": path}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/amapi/run-v3", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("apitoken", c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("amapi/run-v3 status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, err
	}
	return root, nil
}

func normalizeArtworkURL(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	replaced := strings.ReplaceAll(raw, "{w}", "600")
	replaced = strings.ReplaceAll(replaced, "{h}", "600")
	replaced = strings.ReplaceAll(replaced, "{f}", "jpg")
	return replaced
}

func parseRunV3DataAndNext(root map[string]any) ([]map[string]any, string) {
	if root == nil {
		return nil, ""
	}

	container := root
	if dataObj, ok := root["data"].(map[string]any); ok {
		container = dataObj
	}

	raw, _ := container["data"].([]any)
	rows := make([]map[string]any, 0, len(raw))
	for _, it := range raw {
		if m, ok := it.(map[string]any); ok {
			rows = append(rows, m)
		}
	}
	next, _ := container["next"].(string)
	return rows, next
}

func extractMapAny(m map[string]any, path ...string) any {
	var cur any = m
	for _, p := range path {
		switch node := cur.(type) {
		case map[string]any:
			cur = node[p]
		case []any:
			if p == "0" && len(node) > 0 {
				cur = node[0]
			} else {
				return nil
			}
		default:
			return nil
		}
	}
	return cur
}

func extractMapString(m map[string]any, path ...string) string {
	v := extractMapAny(m, path...)
	s, _ := v.(string)
	return s
}

func anyToInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		return int64(n), true
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		if err == nil {
			return i, true
		}
	case json.Number:
		i, err := n.Int64()
		if err == nil {
			return i, true
		}
	}
	return 0, false
}
