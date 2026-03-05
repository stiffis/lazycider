# lazycider

Terminal UI client for Cider (Apple Music) built with Go and Bubble Tea.

## Features

- Three-pane layout (left navigation, center content, right now-playing/lyrics)
- Cider integration for playback, queue, playlists, volume, shuffle, repeat, autoplay
- Search mode (`Ctrl+s`) with grouped modules (`Songs`, `Artists`, `Albums`, `Playlists`)
- Search detail pane for non-song results (artist/album/playlist detail)
- Collapsible modules in search results (`h`/`l` or `Enter` on module headers)
- Lyrics view with provider fallback chain:
  1. Cider local API
  2. LRCLIB
  3. lyrics.ovh
- Local state persistence in `~/.config/lazycider/state.json`

## Requirements

- Go 1.25+
- Running Cider client with RPC enabled

Optional:

- Kitty terminal (for inline cover art rendering)

## Configuration

Environment variables:

- `CIDER_API_BASE` (default: `http://localhost:10767`)
- `CIDER_API_TOKEN` (required if Cider RPC auth is enabled)
- `CIDER_STOREFRONT` (default: `us`, used for catalog search)

## Build and Run

```bash
go mod tidy
go build ./...
go run ./cmd/lazycider
```

## Keybindings

### Global

- `Ctrl+c`: quit
- `:`: command mode (`:q` to quit)

### Focus

- `Ctrl+h` / `Ctrl+k`: move focus left
- `Ctrl+l` / `Ctrl+j`: move focus right

When search is active, focus order is:

`left -> center results -> center detail -> right`

### Navigation

- `j` / `k` (or arrows): move selection/scroll in focused panel
- `gg`: jump to top
- `G`: jump to bottom

### Playback

- `Enter` on a song row: play selected track
- `Space`: play/pause
- `n` / `p`: next/previous
- `+` / `-`: volume up/down
- `s`: toggle shuffle
- `e`: toggle repeat
- `a`: toggle autoplay

### Right Panel

- `y`: toggle queue/lyrics subpanel
- `r`: refresh now playing

### Search

- `Ctrl+s`: open search input (top-left bar)
- `Enter` in search mode: execute query
- `Esc` in search mode: cancel
- `h` / `l` in center results: collapse/expand result modules
- `Enter` on non-song result: load detail in center detail pane

## Security Notes

- Persisted state is stored with restrictive permissions (`0700` directory, `0600` file)
- Display text is sanitized before rendering to reduce terminal control-sequence injection risk
- Artwork downloads are restricted to `http/https` and size-limited

## Project Structure

```text
cmd/lazycider/        # entrypoint
internal/cider/       # Cider API client (playback, library, lyrics, search)
internal/tui/         # Bubble Tea model/update/view logic
internal/artwork/     # cover cache/downloader
internal/term/kitty/  # kitty image rendering helpers
internal/music/       # shared domain types
```

## Development

```bash
gofmt -w ./...
go test ./...
go build ./...
```
