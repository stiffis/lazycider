package artwork

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Cache(raw string) (string, int, int, error) {
	if strings.TrimSpace(raw) == "" {
		return "", 0, 0, fmt.Errorf("empty artwork url")
	}

	client := &http.Client{Timeout: 12 * time.Second}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", 0, 0, err
	}

	ext := strings.ToLower(filepath.Ext(parsed.Path))
	if ext == "" {
		ext = ".jpg"
	}

	h := sha1.Sum([]byte(raw))
	filename := hex.EncodeToString(h[:]) + ext
	cacheDir := filepath.Join(os.TempDir(), "lazycider-cover-cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", 0, 0, err
	}

	local := filepath.Join(cacheDir, filename)
	if _, err := os.Stat(local); err != nil {
		req, err := http.NewRequest(http.MethodGet, raw, nil)
		if err != nil {
			return "", 0, 0, err
		}
		req.Header.Set("User-Agent", "lazycider/0.1")

		resp, err := client.Do(req)
		if err != nil {
			return "", 0, 0, err
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
			return "", 0, 0, fmt.Errorf("artwork http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		tmp := local + ".tmp"
		f, err := os.Create(tmp)
		if err != nil {
			return "", 0, 0, err
		}
		if _, err = io.Copy(f, resp.Body); err != nil {
			f.Close()
			_ = os.Remove(tmp)
			return "", 0, 0, err
		}
		if err = f.Close(); err != nil {
			_ = os.Remove(tmp)
			return "", 0, 0, err
		}
		if err = os.Rename(tmp, local); err != nil {
			_ = os.Remove(tmp)
			return "", 0, 0, err
		}
	}

	f, err := os.Open(local)
	if err != nil {
		return "", 0, 0, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return "", 0, 0, err
	}

	return local, cfg.Width, cfg.Height, nil
}
