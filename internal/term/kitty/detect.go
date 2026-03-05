package kitty

import (
	"os"
	"strings"
)

func IsKitty() bool {
	return strings.Contains(os.Getenv("TERM"), "kitty") || strings.TrimSpace(os.Getenv("KITTY_WINDOW_ID")) != ""
}

func UseUnicodePlaceholders() bool {
	if strings.TrimSpace(os.Getenv("LAZYCIDER_FORCE_UNICODE_PLACEHOLDER")) == "1" {
		return true
	}
	if strings.TrimSpace(os.Getenv("TMUX")) != "" {
		return true
	}
	if strings.TrimSpace(os.Getenv("STY")) != "" {
		return true
	}
	return false
}
