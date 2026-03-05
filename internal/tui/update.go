package tui

import (
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchCoverCmd(m.cider),
		coverTickCmd(),
		fetchPlaybackCmd(m.cider),
		playbackTickCmd(),
		fetchPlaylistsCmd(m.cider),
		fetchQueueCmd(m.cider, m.trackID),
		queueTickCmd(),
		loadStateCmd(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		l := m.layoutInfo()
		m.ensureUpNextViewport(upNextVisibleItems(l.rightQueueHeight))
		lyricsLines := m.lyricsBodyLines(l.rightWidth)
		m.ensureLyricsViewport(lyricsVisibleItems(l.rightQueueHeight), len(lyricsLines))
		m.ensureLeftViewport(leftVisibleItems(l.panelHeight))
		m.ensureCenterViewport(centerVisibleItems(l.panelHeight))
		m.ensureSearchDetailViewport(searchDetailVisibleItems(l.panelHeight))
		return m, m.drawCoverCmd(true)

	case coverTickMsg:
		return m, tea.Batch(fetchCoverCmd(m.cider), coverTickCmd())

	case playbackTickMsg:
		return m, tea.Batch(fetchPlaybackCmd(m.cider), playbackTickCmd())

	case queueTickMsg:
		return m, tea.Batch(fetchQueueCmd(m.cider, m.trackID), queueTickCmd())

	case appStateLoadedMsg:
		if msg.err != nil {
			if os.IsNotExist(msg.err) {
				return m, nil
			}
			return m, nil
		}
		m.activePlaylistID = strings.TrimSpace(msg.state.ActivePlaylistID)
		m.activePlaylistName = strings.TrimSpace(msg.state.ActivePlaylistName)
		if m.activePlaylistName != "" && strings.TrimSpace(msg.state.ActivePlaylistURL) != "" {
			m.playlistURLByName[m.activePlaylistName] = strings.TrimSpace(msg.state.ActivePlaylistURL)
		}
		if strings.TrimSpace(msg.state.CurrentTrackID) != "" {
			m.trackID = strings.TrimSpace(msg.state.CurrentTrackID)
		}
		if len(msg.state.ContextTrackIDs) > 0 {
			restored := make([]centerSongRow, 0, len(msg.state.ContextTrackIDs))
			for i, id := range msg.state.ContextTrackIDs {
				id = strings.TrimSpace(id)
				if id == "" {
					continue
				}
				restored = append(restored, centerSongRow{ID: id, Title: "Track " + strconv.Itoa(i+1), Artist: "Restored context"})
			}
			if len(restored) > 0 {
				m.centerSongs = restored
				m.centerSelected = msg.state.CenterSelected
				l := m.layoutInfo()
				m.ensureCenterViewport(centerVisibleItems(l.panelHeight))
			}
		}
		if m.activePlaylistName != "" {
			m.centerTitle = "Playlist · " + m.activePlaylistName
		}
		if m.activePlaylistID != "" {
			m.restoreCenterSelected = msg.state.CenterSelected
			return m, fetchPlaylistTracksCmd(m.cider, m.activePlaylistName, m.activePlaylistID)
		}
		return m, nil

	case searchSongsLoadedMsg:
		if msg.err != nil {
			m.searchActive = true
			m.searchQuery = strings.TrimSpace(msg.query)
			m.searchSections = []searchSection{{Title: "Songs (0)", Expanded: true, Songs: nil}, {Title: "Artists (0)", Expanded: true, Songs: nil}, {Title: "Albums (0)", Expanded: true, Songs: nil}, {Title: "Playlists (0)", Expanded: true, Songs: nil}}
			m.rebuildCenterFromSearch()
			m.setSearchDetail("Detail", []string{"Search failed", msg.err.Error()})
			m.centerTitle = "Search · " + msg.query
			m.centerSelected = 0
			m.centerTop = 0
			l := m.layoutInfo()
			m.ensureCenterViewport(centerVisibleItems(l.panelHeight))
			m.ensureSearchDetailViewport(searchDetailVisibleItems(l.panelHeight))
			m.focus = PanelCenter
			return m, nil
		}

		m.searchActive = true
		m.searchQuery = strings.TrimSpace(msg.query)
		m.searchSections = append([]searchSection(nil), msg.sections...)
		m.rebuildCenterFromSearch()
		m.setSearchDetail("Detail", []string{"Select a result and press Enter"})
		m.centerTitle = "Search · " + msg.query
		m.centerSelected = 0
		if len(m.centerSongs) > 1 {
			m.centerSelected = 1
		}
		m.centerTop = 0
		l := m.layoutInfo()
		m.ensureCenterViewport(centerVisibleItems(l.panelHeight))
		m.ensureSearchDetailViewport(searchDetailVisibleItems(l.panelHeight))
		m.focus = PanelCenter
		return m, nil

	case searchDetailLoadedMsg:
		if msg.err != nil {
			m.setSearchDetail("Detail", []string{"Failed to load detail", msg.err.Error()})
			l := m.layoutInfo()
			m.ensureSearchDetailViewport(searchDetailVisibleItems(l.panelHeight))
			return m, nil
		}
		m.setSearchDetail(msg.title, msg.lines)
		l := m.layoutInfo()
		m.ensureSearchDetailViewport(searchDetailVisibleItems(l.panelHeight))
		m.focus = PanelCenterDetail
		return m, nil

	case playbackLoadedMsg:
		prevPlaying := m.playing
		prevTrackID := strings.TrimSpace(m.trackID)
		prevProgress := m.progress
		if msg.err == nil {
			m.ciderConnected = true
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
			m.shuffle = msg.shuffleMode != 0
			m.repeat = msg.repeatMode
			if msg.autoplayKnown {
				m.autoplay = msg.autoplay
				m.autoplayKnown = true
			}
			if msg.playingKnown {
				m.playing = msg.playing
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

			trackChanged := strings.TrimSpace(msg.trackID) != "" && strings.TrimSpace(msg.trackID) != prevTrackID
			if trackChanged && m.rightPanelMode == RightPanelLyrics {
				m.lyricsText = ""
				m.lyricsErr = ""
				m.lyricsTop = 0
				return m, fetchLyricsCmd(m.cider, m.trackID, m.track, m.artist, m.album)
			}

			ended := false
			if msg.playingKnown && prevPlaying && !m.playing {
				if strings.TrimSpace(msg.trackID) != "" && strings.TrimSpace(msg.trackID) == prevTrackID {
					if msg.totalSec > 0 && msg.currentSec >= msg.totalSec-0.7 {
						ended = true
					} else if prevProgress >= 0.98 {
						ended = true
					}
				}
			}
			if ended {
				if id, idx, ok := m.adjacentCenterTrackID(1); ok {
					m.centerSelected = idx
					l := m.layoutInfo()
					m.ensureCenterViewport(centerVisibleItems(l.panelHeight))
					return m, playTrackCmd(m.cider, id)
				}
			}
		} else {
			m.ciderConnected = false
		}
		return m, nil

	case playItemResultMsg:
		if msg.err == nil && strings.TrimSpace(msg.trackID) != "" {
			m.ciderConnected = true
			m.trackID = strings.TrimSpace(msg.trackID)
			return m, tea.Batch(fetchPlaybackCmd(m.cider), fetchCoverCmd(m.cider), fetchQueueCmd(m.cider, m.trackID), saveStateCmd(m.snapshotState()))
		}
		if msg.err != nil {
			m.ciderConnected = false
		}
		return m, nil

	case queueLoadedMsg:
		if msg.err == nil {
			m.ciderConnected = true
			m.upNext = append([]upNextRow(nil), msg.items...)
			m.upNextSelected = 0
			m.upNextTop = 0
			l := m.layoutInfo()
			m.ensureUpNextViewport(upNextVisibleItems(l.rightQueueHeight))
		} else {
			m.ciderConnected = false
		}
		return m, nil

	case playbackControlMsg:
		if msg.err == nil {
			m.ciderConnected = true
			if msg.setVolume {
				m.volume = msg.volume
			}
			return m, tea.Batch(fetchPlaybackCmd(m.cider), fetchQueueCmd(m.cider, m.trackID))
		}
		m.ciderConnected = false
		return m, nil

	case playlistsLoadedMsg:
		if msg.err == nil {
			m.ciderConnected = true
			for i := range m.leftModules {
				if m.leftModules[i].Name == "Playlists" {
					m.leftModules[i].Items = append([]string(nil), msg.names...)
					break
				}
			}
			for name, id := range msg.ids {
				m.playlistIDByName[name] = id
			}
			for name, u := range msg.urls {
				m.playlistURLByName[name] = u
			}
			l := m.layoutInfo()
			m.ensureLeftViewport(leftVisibleItems(l.panelHeight))
			if m.activePlaylistName != "" && m.activePlaylistID == "" {
				m.activePlaylistID = strings.TrimSpace(m.playlistIDByName[m.activePlaylistName])
			}
			return m, saveStateCmd(m.snapshotState())
		} else {
			m.ciderConnected = false
		}
		return m, nil

	case playlistTracksLoadedMsg:
		m.clearSearchContext()
		if msg.err != nil {
			m.ciderConnected = false
			m.centerSongs = []centerSongRow{{Title: "No tracks found", Artist: msg.name, Duration: ""}}
		} else {
			m.ciderConnected = true
			m.centerSongs = append([]centerSongRow(nil), msg.songs...)
			m.playlistCache[msg.name] = append([]centerSongRow(nil), msg.songs...)
		}
		m.centerSelected = 0
		m.centerTop = 0
		l := m.layoutInfo()
		m.ensureCenterViewport(centerVisibleItems(l.panelHeight))
		if m.restoreCenterSelected >= 0 && len(m.centerSongs) > 0 {
			if m.restoreCenterSelected >= len(m.centerSongs) {
				m.centerSelected = len(m.centerSongs) - 1
			} else {
				m.centerSelected = m.restoreCenterSelected
			}
			m.restoreCenterSelected = -1
			m.ensureCenterViewport(centerVisibleItems(l.panelHeight))
		}
		return m, tea.Batch(fetchQueueCmd(m.cider, m.trackID), saveStateCmd(m.snapshotState()))

	case coverLoadedMsg:
		if msg.err != nil {
			m.ciderConnected = false
			if m.coverPath == "" {
				m.coverErr = msg.err.Error()
			}
			return m, nil
		}
		m.ciderConnected = true

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
		if trackChanged {
			m.lyricsTop = 0
		}
		m.coverErr = ""

		l := m.layoutInfo()
		m.ensureUpNextViewport(upNextVisibleItems(l.rightQueueHeight))

		if coverChanged {
			m.lastCoverKey = ""
		}

		if m.rightPanelMode == RightPanelLyrics && (trackChanged || m.lyricsText == "") {
			return m, tea.Batch(m.drawCoverCmd(false), fetchLyricsCmd(m.cider, m.trackID, m.track, m.artist, m.album))
		}

		return m, m.drawCoverCmd(false)

	case lyricsLoadedMsg:
		if msg.err != nil {
			m.lyricsErr = msg.err.Error()
			m.lyricsTop = 0
			return m, nil
		}
		m.lyricsText = msg.text
		m.lyricsErr = ""
		m.lyricsTop = 0
		l := m.layoutInfo()
		lyricsLines := m.lyricsBodyLines(l.rightWidth)
		m.ensureLyricsViewport(lyricsVisibleItems(l.rightQueueHeight), len(lyricsLines))
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
			case "ctrl+s":
				m.searchPrevFocus = m.focus
				m.state = StateSearch
				m.cmdInput = ""
				m.focus = PanelLeft
			case "ctrl+c":
				_ = savePersistedState(m.snapshotState())
				return m, tea.Quit
			case "ctrl+h", "ctrl+k":
				switch m.focus {
				case PanelRight:
					if m.searchActive {
						m.focus = PanelCenterDetail
					} else {
						m.focus = PanelCenter
					}
				case PanelCenterDetail:
					m.focus = PanelCenter
				case PanelCenter:
					m.focus = PanelLeft
				}
			case "ctrl+l", "ctrl+j":
				switch m.focus {
				case PanelLeft:
					m.focus = PanelCenter
				case PanelCenter:
					if m.searchActive {
						m.focus = PanelCenterDetail
					} else {
						m.focus = PanelRight
					}
				case PanelCenterDetail:
					m.focus = PanelRight
				}
			case "j", "down":
				if m.focus == PanelLeft {
					l := m.layoutInfo()
					m.moveLeftSelection(1, leftVisibleItems(l.panelHeight))
				} else if m.focus == PanelCenter {
					l := m.layoutInfo()
					m.moveCenterSelection(1, centerVisibleItems(l.panelHeight))
				} else if m.focus == PanelCenterDetail && m.searchActive {
					l := m.layoutInfo()
					m.moveSearchDetail(1, searchDetailVisibleItems(l.panelHeight))
				} else if m.focus == PanelRight && m.rightPanelMode == RightPanelCover {
					l := m.layoutInfo()
					m.moveUpNextSelection(1, upNextVisibleItems(l.rightQueueHeight))
				} else if m.focus == PanelRight && m.rightPanelMode == RightPanelLyrics {
					l := m.layoutInfo()
					lyricsLines := m.lyricsBodyLines(l.rightWidth)
					m.moveLyricsScroll(1, lyricsVisibleItems(l.rightQueueHeight), len(lyricsLines))
				}
			case "k", "up":
				if m.focus == PanelLeft {
					l := m.layoutInfo()
					m.moveLeftSelection(-1, leftVisibleItems(l.panelHeight))
				} else if m.focus == PanelCenter {
					l := m.layoutInfo()
					m.moveCenterSelection(-1, centerVisibleItems(l.panelHeight))
				} else if m.focus == PanelCenterDetail && m.searchActive {
					l := m.layoutInfo()
					m.moveSearchDetail(-1, searchDetailVisibleItems(l.panelHeight))
				} else if m.focus == PanelRight && m.rightPanelMode == RightPanelCover {
					l := m.layoutInfo()
					m.moveUpNextSelection(-1, upNextVisibleItems(l.rightQueueHeight))
				} else if m.focus == PanelRight && m.rightPanelMode == RightPanelLyrics {
					l := m.layoutInfo()
					lyricsLines := m.lyricsBodyLines(l.rightWidth)
					m.moveLyricsScroll(-1, lyricsVisibleItems(l.rightQueueHeight), len(lyricsLines))
				}
			case "enter":
				if m.focus == PanelLeft {
					if name, ok := m.selectedPlaylistName(); ok {
						m.clearSearchContext()
						m.activePlaylistID = strings.TrimSpace(m.playlistIDByName[name])
						m.activePlaylistName = name
						m.focus = PanelCenter
						m.centerTitle = "Playlist · " + name
						m.centerSongs = []centerSongRow{{Title: "Loading playlist...", Artist: name, Duration: ""}}
						m.centerSelected = 0
						m.centerTop = 0
						if cached, ok := m.playlistCache[name]; ok && len(cached) > 0 {
							m.centerSongs = append([]centerSongRow(nil), cached...)
							l := m.layoutInfo()
							m.ensureCenterViewport(centerVisibleItems(l.panelHeight))
							return m, saveStateCmd(m.snapshotState())
						}
						id := strings.TrimSpace(m.playlistIDByName[name])
						return m, fetchPlaylistTracksCmd(m.cider, name, id)
					}
					l := m.layoutInfo()
					m.toggleLeftModuleAtSelection(leftVisibleItems(l.panelHeight))
				} else if m.focus == PanelCenter {
					if m.searchActive && len(m.centerSongs) > 0 && m.centerSelected >= 0 && m.centerSelected < len(m.centerSongs) {
						if m.centerSongs[m.centerSelected].IsModule {
							l := m.layoutInfo()
							m.toggleCenterSearchModuleAtSelection(centerVisibleItems(l.panelHeight))
							return m, nil
						}
						row := m.centerSongs[m.centerSelected]
						kind := strings.TrimSpace(row.Kind)
						if kind != "" && kind != "songs" && kind != "library-songs" {
							m.setSearchDetail("Loading...", []string{"Fetching " + kind + " details..."})
							return m, fetchSearchDetailCmd(m.cider, kind, row.ID)
						}
					}
					if id, ok := m.selectedCenterTrackID(); ok {
						return m, playTrackCmd(m.cider, id)
					}
				}
			case "l":
				if m.focus == PanelLeft {
					if name, ok := m.selectedPlaylistName(); ok {
						m.clearSearchContext()
						m.activePlaylistID = strings.TrimSpace(m.playlistIDByName[name])
						m.activePlaylistName = name
						m.focus = PanelCenter
						m.centerTitle = "Playlist · " + name
						m.centerSongs = []centerSongRow{{Title: "Loading playlist...", Artist: name, Duration: ""}}
						m.centerSelected = 0
						m.centerTop = 0
						if cached, ok := m.playlistCache[name]; ok && len(cached) > 0 {
							m.centerSongs = append([]centerSongRow(nil), cached...)
							l := m.layoutInfo()
							m.ensureCenterViewport(centerVisibleItems(l.panelHeight))
							return m, saveStateCmd(m.snapshotState())
						}
						id := strings.TrimSpace(m.playlistIDByName[name])
						return m, fetchPlaylistTracksCmd(m.cider, name, id)
					}
					l := m.layoutInfo()
					m.expandLeftModuleAtSelection(leftVisibleItems(l.panelHeight))
				}
				if m.focus == PanelCenter && m.searchActive {
					l := m.layoutInfo()
					m.setCenterSearchModuleExpandedAtSelection(true, centerVisibleItems(l.panelHeight))
				}
			case "h":
				if m.focus == PanelLeft {
					l := m.layoutInfo()
					m.collapseLeftModuleAtSelection(leftVisibleItems(l.panelHeight))
				}
				if m.focus == PanelCenter && m.searchActive {
					l := m.layoutInfo()
					m.setCenterSearchModuleExpandedAtSelection(false, centerVisibleItems(l.panelHeight))
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
				} else if m.focus == PanelCenterDetail && m.searchActive {
					m.searchDetailTop = 0
					m.ensureSearchDetailViewport(searchDetailVisibleItems(l.panelHeight))
				} else if m.focus == PanelRight && m.rightPanelMode == RightPanelCover {
					m.upNextSelected = 0
					m.ensureUpNextViewport(upNextVisibleItems(l.rightQueueHeight))
				} else if m.focus == PanelRight && m.rightPanelMode == RightPanelLyrics {
					m.lyricsTop = 0
					lyricsLines := m.lyricsBodyLines(l.rightWidth)
					m.ensureLyricsViewport(lyricsVisibleItems(l.rightQueueHeight), len(lyricsLines))
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
				} else if m.focus == PanelCenterDetail && m.searchActive {
					m.searchDetailTop = len(m.searchDetailLines)
					m.ensureSearchDetailViewport(searchDetailVisibleItems(l.panelHeight))
				} else if m.focus == PanelRight && m.rightPanelMode == RightPanelCover {
					if len(m.upNext) > 0 {
						m.upNextSelected = len(m.upNext) - 1
						m.ensureUpNextViewport(upNextVisibleItems(l.rightQueueHeight))
					}
				} else if m.focus == PanelRight && m.rightPanelMode == RightPanelLyrics {
					lyricsLines := m.lyricsBodyLines(l.rightWidth)
					m.ensureLyricsViewport(lyricsVisibleItems(l.rightQueueHeight), len(lyricsLines))
					m.lyricsTop = len(lyricsLines)
					m.ensureLyricsViewport(lyricsVisibleItems(l.rightQueueHeight), len(lyricsLines))
				}
			case "r":
				if m.rightPanelMode == RightPanelLyrics {
					return m, tea.Batch(fetchCoverCmd(m.cider), fetchLyricsCmd(m.cider, m.trackID, m.track, m.artist, m.album), fetchQueueCmd(m.cider, m.trackID))
				}
				return m, tea.Batch(fetchCoverCmd(m.cider), fetchQueueCmd(m.cider, m.trackID))
			case "n":
				if id, idx, ok := m.adjacentCenterTrackID(1); ok {
					m.centerSelected = idx
					l := m.layoutInfo()
					m.ensureCenterViewport(centerVisibleItems(l.panelHeight))
					return m, playTrackCmd(m.cider, id)
				}
				return m, nextCmd(m.cider)
			case "p":
				if id, idx, ok := m.adjacentCenterTrackID(-1); ok {
					m.centerSelected = idx
					l := m.layoutInfo()
					m.ensureCenterViewport(centerVisibleItems(l.panelHeight))
					return m, playTrackCmd(m.cider, id)
				}
				return m, previousCmd(m.cider)
			case " ":
				return m, playPauseCmd(m.cider)
			case "+", "=":
				return m, adjustVolumeCmd(m.cider, m.volume, 5)
			case "-", "_":
				return m, adjustVolumeCmd(m.cider, m.volume, -5)
			case "s":
				return m, toggleShuffleCmd(m.cider)
			case "e":
				return m, toggleRepeatCmd(m.cider)
			case "a":
				return m, toggleAutoplayCmd(m.cider)
			case "y":
				if m.rightPanelMode == RightPanelLyrics {
					m.rightPanelMode = RightPanelCover
					m.lyricsErr = ""
					l := m.layoutInfo()
					m.ensureUpNextViewport(upNextVisibleItems(l.rightQueueHeight))
					return m, nil
				}

				m.rightPanelMode = RightPanelLyrics
				m.lyricsText = ""
				m.lyricsErr = ""
				m.lyricsTop = 0
				return m, fetchLyricsCmd(m.cider, m.trackID, m.track, m.artist, m.album)
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
		case StateSearch:
			switch msg.String() {
			case "enter":
				q := strings.TrimSpace(m.cmdInput)
				m.state = StateNormal
				m.cmdInput = ""
				m.focus = m.searchPrevFocus
				return m, searchSongsCmd(m.cider, q)
			case "esc":
				m.state = StateNormal
				m.cmdInput = ""
				m.focus = m.searchPrevFocus
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
