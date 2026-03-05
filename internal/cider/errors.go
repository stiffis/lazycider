package cider

import "errors"

var (
	ErrMissingTrackID = errors.New("missing track id")
	ErrEmptyLyrics    = errors.New("empty lyrics")
)
