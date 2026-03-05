package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type persistedState struct {
	SchemaVersion      int      `json:"schemaVersion"`
	SavedAt            string   `json:"savedAt"`
	ActivePlaylistID   string   `json:"activePlaylistId"`
	ActivePlaylistName string   `json:"activePlaylistName"`
	ActivePlaylistURL  string   `json:"activePlaylistUrl"`
	CurrentTrackID     string   `json:"currentTrackId"`
	CenterSelected     int      `json:"centerSelected"`
	ContextTrackIDs    []string `json:"contextTrackIds"`
}

func loadStateCmd() tea.Cmd {
	return func() tea.Msg {
		st, err := loadPersistedState()
		return appStateLoadedMsg{state: st, err: err}
	}
}

func saveStateCmd(st persistedState) tea.Cmd {
	return func() tea.Msg {
		_ = savePersistedState(st)
		return nil
	}
}

func (m Model) snapshotState() persistedState {
	ids := make([]string, 0, len(m.centerSongs))
	for _, s := range m.centerSongs {
		id := strings.TrimSpace(s.ID)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return persistedState{
		SchemaVersion:      1,
		SavedAt:            time.Now().UTC().Format(time.RFC3339),
		ActivePlaylistID:   strings.TrimSpace(m.activePlaylistID),
		ActivePlaylistName: strings.TrimSpace(m.activePlaylistName),
		ActivePlaylistURL:  strings.TrimSpace(m.playlistURLByName[m.activePlaylistName]),
		CurrentTrackID:     strings.TrimSpace(m.trackID),
		CenterSelected:     m.centerSelected,
		ContextTrackIDs:    ids,
	}
}

func stateFilePath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "lazycider", "state.json"), nil
}

func loadPersistedState() (persistedState, error) {
	path, err := stateFilePath()
	if err != nil {
		return persistedState{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return persistedState{}, err
	}
	var st persistedState
	if err := json.Unmarshal(b, &st); err != nil {
		return persistedState{}, err
	}
	if st.CenterSelected < 0 {
		st.CenterSelected = 0
	}
	return st, nil
}

func savePersistedState(st persistedState) error {
	path, err := stateFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
