package music

type NowPlaying struct {
	TrackID      string
	Track        string
	Artist       string
	Album        string
	ArtworkURL   string
	DurationMS   int64
	CurrentSec   float64
	RemainingSec float64
}

type Playlist struct {
	ID   string
	Name string
	URL  string
}

type Track struct {
	ID         string
	Title      string
	Artist     string
	Album      string
	URL        string
	DurationMS int64
}
