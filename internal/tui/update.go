package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchCoverCmd(m.cider), coverTickCmd(), fetchPlaybackCmd(m.cider), playbackTickCmd(), fetchPlaylistsCmd(m.cider))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		l := m.layoutInfo()
		m.ensureUpNextViewport(upNextVisibleItems(l.rightQueueHeight))
		m.ensureLeftViewport(leftVisibleItems(l.panelHeight))
		m.ensureCenterViewport(centerVisibleItems(l.panelHeight))
		return m, m.drawCoverCmd(true)

	case coverTickMsg:
		return m, tea.Batch(fetchCoverCmd(m.cider), coverTickCmd())

	case playbackTickMsg:
		return m, tea.Batch(fetchPlaybackCmd(m.cider), playbackTickCmd())

	case playbackLoadedMsg:
		if msg.err == nil {
			m.trackID = strings.TrimSpace(msg.trackID)
			if strings.TrimSpace(msg.track) != "" {
				m.track = strings.TrimSpace(msg.track)
			}
			if strings.TrimSpace(msg.artist) != "" {
				m.artist = strings.TrimSpace(msg.artist)
			}
			if strings.TrimSpace(msg.album) != "" {
				m.album = strings.TrimSpace(msg.album)
			}
			if msg.valid {
				if msg.current != "" {
					m.current = msg.current
				}
				if msg.total != "" {
					m.total = msg.total
				}
				m.progress = msg.progress
			}
		}
		return m, nil

	case playItemResultMsg:
		if msg.err == nil && strings.TrimSpace(msg.trackID) != "" {
			m.trackID = strings.TrimSpace(msg.trackID)
			return m, tea.Batch(fetchPlaybackCmd(m.cider), fetchCoverCmd(m.cider))
		}
		return m, nil

	case playlistsLoadedMsg:
		if msg.err == nil {
			for i := range m.leftModules {
				if m.leftModules[i].Name == "Playlists" {
					m.leftModules[i].Items = append([]string(nil), msg.names...)
					break
				}
			}
			for name, id := range msg.ids {
				m.playlistIDByName[name] = id
			}
			l := m.layoutInfo()
			m.ensureLeftViewport(leftVisibleItems(l.panelHeight))
		}
		return m, nil

	case playlistTracksLoadedMsg:
		if msg.err != nil {
			m.centerSongs = []centerSongRow{{Title: "No tracks found", Artist: msg.name, Duration: ""}}
		} else {
			m.centerSongs = append([]centerSongRow(nil), msg.songs...)
			m.playlistCache[msg.name] = append([]centerSongRow(nil), msg.songs...)
		}
		m.centerSelected = 0
		m.centerTop = 0
		l := m.layoutInfo()
		m.ensureCenterViewport(centerVisibleItems(l.panelHeight))
		return m, nil

	case coverLoadedMsg:
		if msg.err != nil {
			if m.coverPath == "" {
				m.coverErr = msg.err.Error()
			}
			return m, nil
		}

		trackChanged := msg.trackID != "" && msg.trackID != m.trackID

		if msg.track != "" {
			m.track = msg.track
		}
		if msg.artist != "" {
			m.artist = msg.artist
		}
		if msg.album != "" {
			m.album = msg.album
		}

		coverChanged := msg.coverURL != "" && msg.coverURL != m.coverURL
		m.coverURL = msg.coverURL
		m.coverPath = msg.coverPath
		m.coverW = msg.coverW
		m.coverH = msg.coverH
		if msg.trackID != "" {
			m.trackID = msg.trackID
		}
		m.coverErr = ""

		l := m.layoutInfo()
		m.ensureUpNextViewport(upNextVisibleItems(l.rightQueueHeight))

		if coverChanged {
			m.lastCoverKey = ""
		}

		if m.rightPanelMode == RightPanelLyrics {
			if trackChanged || m.lyricsText == "" {
				return m, fetchLyricsCmd(m.cider, m.trackID, m.track, m.artist)
			}
			return m, nil
		}

		return m, m.drawCoverCmd(false)

	case lyricsLoadedMsg:
		if msg.err != nil {
			m.lyricsErr = msg.err.Error()
			return m, nil
		}
		m.lyricsText = msg.text
		m.lyricsErr = ""
		return m, nil

	case coverDrawnMsg:
		if msg.err != nil {
			m.coverErr = msg.err.Error()
			return m, nil
		}
		m.coverErr = ""
		m.lastCoverKey = msg.drawKey
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case StateNormal:
			key := msg.String()
			if key != "g" {
				m.pendingG = false
			}
			switch key {
			case ":":
				m.state = StateCommand
				m.cmdInput = ":"
			case "ctrl+c":
				return m, tea.Quit
			case "ctrl+h", "ctrl+k":
				if m.focus > PanelLeft {
					m.focus--
				}
			case "ctrl+l", "ctrl+j":
				if m.focus < PanelRight {
					m.focus++
				}
			case "j", "down":
				if m.focus == PanelLeft {
					l := m.layoutInfo()
					m.moveLeftSelection(1, leftVisibleItems(l.panelHeight))
				} else if m.focus == PanelCenter {
					l := m.layoutInfo()
					m.moveCenterSelection(1, centerVisibleItems(l.panelHeight))
				} else if m.focus == PanelRight && m.rightPanelMode == RightPanelCover {
					l := m.layoutInfo()
					m.moveUpNextSelection(1, upNextVisibleItems(l.rightQueueHeight))
				}
			case "k", "up":
				if m.focus == PanelLeft {
					l := m.layoutInfo()
					m.moveLeftSelection(-1, leftVisibleItems(l.panelHeight))
				} else if m.focus == PanelCenter {
					l := m.layoutInfo()
					m.moveCenterSelection(-1, centerVisibleItems(l.panelHeight))
				} else if m.focus == PanelRight && m.rightPanelMode == RightPanelCover {
					l := m.layoutInfo()
					m.moveUpNextSelection(-1, upNextVisibleItems(l.rightQueueHeight))
				}
			case "enter":
				if m.focus == PanelLeft {
					if name, ok := m.selectedPlaylistName(); ok {
						m.focus = PanelCenter
						m.centerTitle = "Playlist · " + name
						m.centerSongs = []centerSongRow{{Title: "Loading playlist...", Artist: name, Duration: ""}}
						m.centerSelected = 0
						m.centerTop = 0
						if cached, ok := m.playlistCache[name]; ok && len(cached) > 0 {
							m.centerSongs = append([]centerSongRow(nil), cached...)
							l := m.layoutInfo()
							m.ensureCenterViewport(centerVisibleItems(l.panelHeight))
							return m, nil
						}
						id := strings.TrimSpace(m.playlistIDByName[name])
						return m, fetchPlaylistTracksCmd(m.cider, name, id)
					}
					l := m.layoutInfo()
					m.toggleLeftModuleAtSelection(leftVisibleItems(l.panelHeight))
				} else if m.focus == PanelCenter {
					if id, ok := m.selectedCenterTrackID(); ok {
						return m, playItemCmd(m.cider, id)
					}
				}
			case "l":
				if m.focus == PanelLeft {
					if name, ok := m.selectedPlaylistName(); ok {
						m.focus = PanelCenter
						m.centerTitle = "Playlist · " + name
						m.centerSongs = []centerSongRow{{Title: "Loading playlist...", Artist: name, Duration: ""}}
						m.centerSelected = 0
						m.centerTop = 0
						if cached, ok := m.playlistCache[name]; ok && len(cached) > 0 {
							m.centerSongs = append([]centerSongRow(nil), cached...)
							l := m.layoutInfo()
							m.ensureCenterViewport(centerVisibleItems(l.panelHeight))
							return m, nil
						}
						id := strings.TrimSpace(m.playlistIDByName[name])
						return m, fetchPlaylistTracksCmd(m.cider, name, id)
					}
					l := m.layoutInfo()
					m.expandLeftModuleAtSelection(leftVisibleItems(l.panelHeight))
				}
			case "h":
				if m.focus == PanelLeft {
					l := m.layoutInfo()
					m.collapseLeftModuleAtSelection(leftVisibleItems(l.panelHeight))
				}
			case "g":
				if !m.pendingG {
					m.pendingG = true
					return m, nil
				}
				m.pendingG = false
				l := m.layoutInfo()
				if m.focus == PanelLeft {
					m.leftSelected = 0
					m.ensureLeftViewport(leftVisibleItems(l.panelHeight))
				} else if m.focus == PanelCenter {
					m.centerSelected = 0
					m.ensureCenterViewport(centerVisibleItems(l.panelHeight))
				} else if m.focus == PanelRight && m.rightPanelMode == RightPanelCover {
					m.upNextSelected = 0
					m.ensureUpNextViewport(upNextVisibleItems(l.rightQueueHeight))
				}
			case "G":
				l := m.layoutInfo()
				if m.focus == PanelLeft {
					rows := m.leftVisibleRows()
					if len(rows) > 0 {
						m.leftSelected = len(rows) - 1
						m.ensureLeftViewport(leftVisibleItems(l.panelHeight))
					}
				} else if m.focus == PanelCenter {
					if len(m.centerSongs) > 0 {
						m.centerSelected = len(m.centerSongs) - 1
						m.ensureCenterViewport(centerVisibleItems(l.panelHeight))
					}
				} else if m.focus == PanelRight && m.rightPanelMode == RightPanelCover {
					if len(m.upNext) > 0 {
						m.upNextSelected = len(m.upNext) - 1
						m.ensureUpNextViewport(upNextVisibleItems(l.rightQueueHeight))
					}
				}
			case "r":
				if m.rightPanelMode == RightPanelLyrics {
					return m, tea.Batch(fetchCoverCmd(m.cider), fetchLyricsCmd(m.cider, m.trackID, m.track, m.artist))
				}
				return m, fetchCoverCmd(m.cider)
			case "y":
				if m.rightPanelMode == RightPanelLyrics {
					m.rightPanelMode = RightPanelCover
					m.lyricsErr = ""
					l := m.layoutInfo()
					m.ensureUpNextViewport(upNextVisibleItems(l.rightQueueHeight))
					return m, m.drawCoverCmd(true)
				}

				m.rightPanelMode = RightPanelLyrics
				m.lyricsText = ""
				m.lyricsErr = ""
				return m, tea.Batch(clearKittyImagesCmd(), fetchLyricsCmd(m.cider, m.trackID, m.track, m.artist))
			}
		case StateCommand:
			switch msg.String() {
			case "enter":
				switch m.cmdInput {
				case ":q":
					return m, tea.Quit
				}
				m.state = StateNormal
				m.cmdInput = ""
			case "esc":
				m.state = StateNormal
				m.cmdInput = ""
			case "backspace":
				if len(m.cmdInput) > 0 {
					r := []rune(m.cmdInput)
					m.cmdInput = string(r[:len(r)-1])
				}
			default:
				if runes := []rune(msg.String()); len(runes) == 1 {
					m.cmdInput += msg.String()
				}
			}
		}
	}
	return m, nil
}
