package tui

import (
	"strings"

	"lazycider/internal/cider"
)

type State int
type RightPanelMode int
type PanelFocus int

type upNextRow struct {
	Title    string
	Subtitle string
}

type leftModule struct {
	Name     string
	Items    []string
	Expanded bool
}

type leftRow struct {
	Text      string
	ModuleIdx int
	ItemIdx   int
	IsModule  bool
}

type centerSongRow struct {
	ID       string
	URL      string
	Title    string
	Artist   string
	Duration string
	IsModule bool
	Section  int
	Item     int
	Kind     string
}

type searchSection struct {
	Title    string
	Expanded bool
	Songs    []centerSongRow
}

const (
	StateNormal State = iota
	StateCommand
	StateQueue
	StateSearch
)

const (
	RightPanelCover RightPanelMode = iota
	RightPanelLyrics
)

const (
	PanelLeft PanelFocus = iota
	PanelCenter
	PanelCenterDetail
	PanelRight
)

type Model struct {
	width    int
	height   int
	state    State
	cmdInput string
	focus    PanelFocus
	pendingG bool

	leftModules  []leftModule
	leftSelected int
	leftTop      int

	centerTitle           string
	centerSongs           []centerSongRow
	centerSelected        int
	centerTop             int
	restoreCenterSelected int
	searchActive          bool
	searchQuery           string
	searchSections        []searchSection
	searchPrevFocus       PanelFocus
	searchDetailTitle     string
	searchDetailLines     []string
	searchDetailTop       int

	playlistIDByName   map[string]string
	playlistURLByName  map[string]string
	playlistCache      map[string][]centerSongRow
	activePlaylistID   string
	activePlaylistName string

	track          string
	artist         string
	album          string
	playing        bool
	progress       float64
	current        string
	total          string
	volume         int
	shuffle        bool
	repeat         int
	autoplay       bool
	autoplayKnown  bool
	trackID        string
	ciderConnected bool

	rightPanelMode RightPanelMode
	lyricsText     string
	lyricsErr      string
	lyricsTop      int

	upNext         []upNextRow
	upNextSelected int
	upNextTop      int

	coverPath    string
	coverURL     string
	coverW       int
	coverH       int
	coverErr     string
	lastCoverKey string

	cider *cider.Client
}

func NewModel() Model {
	return NewModelWithClient(cider.NewFromEnv())
}

func NewModelWithClient(client *cider.Client) Model {
	if client == nil {
		client = cider.NewFromEnv()
	}
	return Model{
		state:                 StateNormal,
		focus:                 PanelCenter,
		leftModules:           seedLeftModules(),
		leftSelected:          0,
		leftTop:               0,
		centerTitle:           "Center Content",
		centerSongs:           []centerSongRow{{Title: "Select a playlist", Artist: "Left panel", Duration: ""}},
		centerSelected:        0,
		centerTop:             0,
		restoreCenterSelected: -1,
		searchActive:          false,
		searchQuery:           "",
		searchSections:        nil,
		searchPrevFocus:       PanelCenter,
		searchDetailTitle:     "Detail",
		searchDetailLines:     []string{"Select a result and press Enter"},
		searchDetailTop:       0,
		playlistIDByName:      make(map[string]string),
		playlistURLByName:     make(map[string]string),
		playlistCache:         make(map[string][]centerSongRow),
		track:                 "Sailing",
		artist:                "Christopher Cross",
		album:                 "Christopher Cross",
		playing:               true,
		progress:              0.55,
		current:               "2:34",
		total:                 "4:41",
		volume:                80,
		shuffle:               false,
		repeat:                0,
		autoplay:              false,
		autoplayKnown:         false,
		ciderConnected:        false,
		rightPanelMode:        RightPanelCover,
		lyricsTop:             0,
		upNext:                nil,
		upNextSelected:        0,
		upNextTop:             0,
		cider:                 client,
	}
}

func seedLeftModules() []leftModule {
	return []leftModule{
		{Name: "Apple Music", Items: []string{"Home", "New", "Radio"}, Expanded: true},
		{Name: "Library", Items: []string{"Recently Added", "Songs", "Albums", "Artists"}, Expanded: true},
		{Name: "Pins", Items: []string{"gym"}, Expanded: true},
		{Name: "Apple Music Playlists", Items: nil, Expanded: false},
		{Name: "Playlists", Items: nil, Expanded: true},
	}
}

func (m Model) focusLabel() string {
	switch m.focus {
	case PanelLeft:
		return "left"
	case PanelCenter:
		return "center"
	case PanelCenterDetail:
		return "detail"
	case PanelRight:
		return "right"
	default:
		return "center"
	}
}

func (m *Model) moveUpNextSelection(delta, visible int) {
	if len(m.upNext) == 0 {
		m.upNextSelected = 0
		m.upNextTop = 0
		return
	}
	if delta == 0 {
		m.ensureUpNextViewport(visible)
		return
	}

	m.upNextSelected += delta
	if m.upNextSelected < 0 {
		m.upNextSelected = 0
	}
	if m.upNextSelected >= len(m.upNext) {
		m.upNextSelected = len(m.upNext) - 1
	}
	m.ensureUpNextViewport(visible)
}

func (m Model) leftVisibleRows() []leftRow {
	rows := make([]leftRow, 0, 48)
	for mi, mod := range m.leftModules {
		rows = append(rows, leftRow{Text: mod.Name, ModuleIdx: mi, ItemIdx: -1, IsModule: true})
		if !mod.Expanded {
			continue
		}
		for ii, item := range mod.Items {
			rows = append(rows, leftRow{Text: item, ModuleIdx: mi, ItemIdx: ii, IsModule: false})
		}
	}
	return rows
}

func (m *Model) ensureLeftViewport(visible int) {
	rows := m.leftVisibleRows()
	if len(rows) == 0 {
		m.leftSelected = 0
		m.leftTop = 0
		return
	}
	if visible < 1 {
		visible = 1
	}
	if m.leftSelected < 0 {
		m.leftSelected = 0
	}
	if m.leftSelected >= len(rows) {
		m.leftSelected = len(rows) - 1
	}

	maxTop := len(rows) - visible
	if maxTop < 0 {
		maxTop = 0
	}
	if m.leftTop < 0 {
		m.leftTop = 0
	}
	if m.leftTop > maxTop {
		m.leftTop = maxTop
	}

	if m.leftSelected < m.leftTop {
		m.leftTop = m.leftSelected
	}
	if m.leftSelected >= m.leftTop+visible {
		m.leftTop = m.leftSelected - visible + 1
	}
	if m.leftTop < 0 {
		m.leftTop = 0
	}
	if m.leftTop > maxTop {
		m.leftTop = maxTop
	}
}

func (m *Model) moveLeftSelection(delta, visible int) {
	rows := m.leftVisibleRows()
	if len(rows) == 0 {
		m.leftSelected = 0
		m.leftTop = 0
		return
	}
	if delta == 0 {
		m.ensureLeftViewport(visible)
		return
	}

	m.leftSelected += delta
	if m.leftSelected < 0 {
		m.leftSelected = 0
	}
	if m.leftSelected >= len(rows) {
		m.leftSelected = len(rows) - 1
	}
	m.ensureLeftViewport(visible)
}

func (m *Model) toggleLeftModuleAtSelection(visible int) {
	rows := m.leftVisibleRows()
	if len(rows) == 0 || m.leftSelected < 0 || m.leftSelected >= len(rows) {
		return
	}
	r := rows[m.leftSelected]
	if !r.IsModule || r.ModuleIdx < 0 || r.ModuleIdx >= len(m.leftModules) {
		return
	}
	m.leftModules[r.ModuleIdx].Expanded = !m.leftModules[r.ModuleIdx].Expanded
	m.ensureLeftViewport(visible)
}

func (m *Model) expandLeftModuleAtSelection(visible int) {
	rows := m.leftVisibleRows()
	if len(rows) == 0 || m.leftSelected < 0 || m.leftSelected >= len(rows) {
		return
	}
	r := rows[m.leftSelected]
	if !r.IsModule || r.ModuleIdx < 0 || r.ModuleIdx >= len(m.leftModules) {
		return
	}
	m.leftModules[r.ModuleIdx].Expanded = true
	m.ensureLeftViewport(visible)
}

func (m *Model) collapseLeftModuleAtSelection(visible int) {
	rows := m.leftVisibleRows()
	if len(rows) == 0 || m.leftSelected < 0 || m.leftSelected >= len(rows) {
		return
	}
	r := rows[m.leftSelected]
	if r.IsModule {
		if r.ModuleIdx >= 0 && r.ModuleIdx < len(m.leftModules) {
			m.leftModules[r.ModuleIdx].Expanded = false
		}
		m.ensureLeftViewport(visible)
		return
	}
	if r.ModuleIdx >= 0 && r.ModuleIdx < len(m.leftModules) {
		m.leftModules[r.ModuleIdx].Expanded = false
		m.leftSelected = m.leftModuleRowIndex(r.ModuleIdx)
	}
	m.ensureLeftViewport(visible)
}

func (m Model) leftModuleRowIndex(moduleIdx int) int {
	if moduleIdx < 0 || moduleIdx >= len(m.leftModules) {
		return 0
	}
	idx := 0
	for i := 0; i < moduleIdx; i++ {
		idx++
		if m.leftModules[i].Expanded {
			idx += len(m.leftModules[i].Items)
		}
	}
	return idx
}

func leftVisibleItems(totalHeight int) int {
	v := totalHeight - 3
	if v < 1 {
		v = 1
	}
	return v
}

func centerVisibleItems(totalHeight int) int {
	v := totalHeight - centerPlayerHeight(totalHeight) - 2
	if v < 1 {
		v = 1
	}
	return v
}

func centerPlayerHeight(totalHeight int) int {
	h := 3
	if totalHeight < h {
		h = totalHeight
	}
	if h < 0 {
		h = 0
	}
	return h
}

func (m *Model) ensureCenterViewport(visible int) {
	if len(m.centerSongs) == 0 {
		m.centerSelected = 0
		m.centerTop = 0
		return
	}
	if visible < 1 {
		visible = 1
	}
	if m.centerSelected < 0 {
		m.centerSelected = 0
	}
	if m.centerSelected >= len(m.centerSongs) {
		m.centerSelected = len(m.centerSongs) - 1
	}

	maxTop := len(m.centerSongs) - visible
	if maxTop < 0 {
		maxTop = 0
	}
	if m.centerTop < 0 {
		m.centerTop = 0
	}
	if m.centerTop > maxTop {
		m.centerTop = maxTop
	}
	if m.centerSelected < m.centerTop {
		m.centerTop = m.centerSelected
	}
	if m.centerSelected >= m.centerTop+visible {
		m.centerTop = m.centerSelected - visible + 1
	}
	if m.centerTop < 0 {
		m.centerTop = 0
	}
	if m.centerTop > maxTop {
		m.centerTop = maxTop
	}
}

func (m *Model) moveCenterSelection(delta, visible int) {
	if len(m.centerSongs) == 0 {
		m.centerSelected = 0
		m.centerTop = 0
		return
	}
	if delta == 0 {
		m.ensureCenterViewport(visible)
		return
	}

	m.centerSelected += delta
	if m.centerSelected < 0 {
		m.centerSelected = 0
	}
	if m.centerSelected >= len(m.centerSongs) {
		m.centerSelected = len(m.centerSongs) - 1
	}
	m.ensureCenterViewport(visible)
}

func (m Model) selectedPlaylistName() (string, bool) {
	rows := m.leftVisibleRows()
	if len(rows) == 0 || m.leftSelected < 0 || m.leftSelected >= len(rows) {
		return "", false
	}
	r := rows[m.leftSelected]
	if r.IsModule || r.ModuleIdx < 0 || r.ModuleIdx >= len(m.leftModules) {
		return "", false
	}
	if m.leftModules[r.ModuleIdx].Name != "Playlists" {
		return "", false
	}
	name := strings.TrimSpace(r.Text)
	if name == "" {
		return "", false
	}
	return name, true
}

func (m Model) selectedCenterTrackID() (string, bool) {
	if len(m.centerSongs) == 0 || m.centerSelected < 0 || m.centerSelected >= len(m.centerSongs) {
		return "", false
	}
	id := strings.TrimSpace(m.centerSongs[m.centerSelected].ID)
	if id == "" {
		return "", false
	}
	return id, true
}

func (m Model) adjacentCenterTrackID(delta int) (string, int, bool) {
	if len(m.centerSongs) == 0 || delta == 0 {
		return "", 0, false
	}

	base := -1
	if strings.TrimSpace(m.trackID) != "" {
		for i, s := range m.centerSongs {
			if strings.TrimSpace(s.ID) != "" && strings.TrimSpace(s.ID) == strings.TrimSpace(m.trackID) {
				base = i
				break
			}
		}
	}
	if base < 0 {
		base = m.centerSelected
	}
	if base < 0 || base >= len(m.centerSongs) {
		return "", 0, false
	}

	step := 1
	if delta < 0 {
		step = -1
	}
	for i := base + step; i >= 0 && i < len(m.centerSongs); i += step {
		id := strings.TrimSpace(m.centerSongs[i].ID)
		if id != "" {
			return id, i, true
		}
	}

	return "", 0, false
}

func (m Model) centerTrackIDsAfterSelection() []string {
	if len(m.centerSongs) == 0 || m.centerSelected < 0 || m.centerSelected >= len(m.centerSongs) {
		return nil
	}
	out := make([]string, 0, len(m.centerSongs)-m.centerSelected-1)
	for i := m.centerSelected + 1; i < len(m.centerSongs); i++ {
		id := strings.TrimSpace(m.centerSongs[i].ID)
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}

func (m *Model) clearSearchContext() {
	m.searchActive = false
	m.searchQuery = ""
	m.searchSections = nil
	m.searchDetailTitle = "Detail"
	m.searchDetailLines = []string{"Select a result and press Enter"}
	m.searchDetailTop = 0
	if m.focus == PanelCenterDetail {
		m.focus = PanelCenter
	}
}

func (m *Model) rebuildCenterFromSearch() {
	rows := make([]centerSongRow, 0, 64)
	for si, sec := range m.searchSections {
		rows = append(rows, centerSongRow{Title: sec.Title, IsModule: true, Section: si, Item: -1})
		if !sec.Expanded {
			continue
		}
		if len(sec.Songs) == 0 {
			rows = append(rows, centerSongRow{Title: "  No matches", Artist: "Try another query", Section: si, Item: -1})
			continue
		}
		for i, s := range sec.Songs {
			row := s
			row.Title = "  " + strings.TrimSpace(row.Title)
			row.IsModule = false
			row.Section = si
			row.Item = i
			if strings.TrimSpace(row.Kind) == "" {
				row.Kind = "songs"
			}
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		rows = []centerSongRow{{Title: "Songs (0)", IsModule: true, Section: 0, Item: -1, Kind: "songs"}}
	}
	m.centerSongs = rows
}

func (m Model) selectedCenterSearchRow() (centerSongRow, bool) {
	if len(m.centerSongs) == 0 || m.centerSelected < 0 || m.centerSelected >= len(m.centerSongs) {
		return centerSongRow{}, false
	}
	return m.centerSongs[m.centerSelected], true
}

func (m *Model) setSearchDetail(title string, lines []string) {
	t := strings.TrimSpace(title)
	if t == "" {
		t = "Detail"
	}
	if len(lines) == 0 {
		lines = []string{"No details available"}
	}
	m.searchDetailTitle = t
	m.searchDetailLines = append([]string(nil), lines...)
	m.searchDetailTop = 0
}

func (m *Model) ensureSearchDetailViewport(visible int) {
	if visible < 1 {
		visible = 1
	}
	total := len(m.searchDetailLines)
	if total < 0 {
		total = 0
	}
	maxTop := total - visible
	if maxTop < 0 {
		maxTop = 0
	}
	if m.searchDetailTop < 0 {
		m.searchDetailTop = 0
	}
	if m.searchDetailTop > maxTop {
		m.searchDetailTop = maxTop
	}
}

func (m *Model) moveSearchDetail(delta, visible int) {
	if delta == 0 {
		m.ensureSearchDetailViewport(visible)
		return
	}
	m.searchDetailTop += delta
	m.ensureSearchDetailViewport(visible)
}

func centerSearchPaneHeights(innerHeight int) (int, int) {
	if innerHeight <= 0 {
		return 0, 0
	}
	top := (innerHeight * 55) / 100
	if top < 6 {
		top = 6
	}
	if innerHeight-top < 6 {
		top = innerHeight - 6
	}
	if top < 1 {
		top = innerHeight
	}
	bottom := innerHeight - top
	if bottom < 0 {
		bottom = 0
	}
	return top, bottom
}

func searchDetailVisibleItems(centerHeight int) int {
	player := centerPlayerHeight(centerHeight)
	inner := centerHeight - player
	if inner < 1 {
		return 1
	}
	_, bottom := centerSearchPaneHeights(inner)
	v := bottom - 2
	if v < 1 {
		v = 1
	}
	return v
}

func (m Model) selectedCenterSection() (int, bool) {
	if len(m.centerSongs) == 0 || m.centerSelected < 0 || m.centerSelected >= len(m.centerSongs) {
		return 0, false
	}
	r := m.centerSongs[m.centerSelected]
	if r.Section < 0 || r.Section >= len(m.searchSections) {
		return 0, false
	}
	return r.Section, true
}

func (m *Model) toggleCenterSearchModuleAtSelection(visible int) {
	if !m.searchActive {
		return
	}
	sectionIdx, ok := m.selectedCenterSection()
	if !ok {
		return
	}
	m.searchSections[sectionIdx].Expanded = !m.searchSections[sectionIdx].Expanded
	m.rebuildCenterFromSearch()
	for i := range m.centerSongs {
		if m.centerSongs[i].IsModule && m.centerSongs[i].Section == sectionIdx {
			m.centerSelected = i
			break
		}
	}
	m.ensureCenterViewport(visible)
}

func (m *Model) setCenterSearchModuleExpandedAtSelection(expanded bool, visible int) {
	if !m.searchActive {
		return
	}
	sectionIdx, ok := m.selectedCenterSection()
	if !ok {
		return
	}
	m.searchSections[sectionIdx].Expanded = expanded
	m.rebuildCenterFromSearch()
	for i := range m.centerSongs {
		if m.centerSongs[i].IsModule && m.centerSongs[i].Section == sectionIdx {
			if expanded && m.centerSelected < len(m.centerSongs)-1 {
				if i+1 < len(m.centerSongs) && !m.centerSongs[i+1].IsModule && m.centerSongs[i+1].Section == sectionIdx {
					m.centerSelected = i + 1
				} else {
					m.centerSelected = i
				}
			} else {
				m.centerSelected = i
			}
			break
		}
	}
	m.ensureCenterViewport(visible)
}

func (m *Model) ensureUpNextViewport(visible int) {
	if len(m.upNext) == 0 {
		m.upNextSelected = 0
		m.upNextTop = 0
		return
	}
	if visible < 1 {
		visible = 1
	}
	if m.upNextSelected < 0 {
		m.upNextSelected = 0
	}
	if m.upNextSelected >= len(m.upNext) {
		m.upNextSelected = len(m.upNext) - 1
	}

	maxTop := len(m.upNext) - visible
	if maxTop < 0 {
		maxTop = 0
	}
	if m.upNextTop < 0 {
		m.upNextTop = 0
	}
	if m.upNextTop > maxTop {
		m.upNextTop = maxTop
	}

	if m.upNextSelected < m.upNextTop {
		m.upNextTop = m.upNextSelected
	}
	if m.upNextSelected >= m.upNextTop+visible {
		m.upNextTop = m.upNextSelected - visible + 1
	}
	if m.upNextTop < 0 {
		m.upNextTop = 0
	}
	if m.upNextTop > maxTop {
		m.upNextTop = maxTop
	}
}

func (m *Model) ensureLyricsViewport(visible, total int) {
	if visible < 1 {
		visible = 1
	}
	if total < 0 {
		total = 0
	}
	maxTop := total - visible
	if maxTop < 0 {
		maxTop = 0
	}
	if m.lyricsTop < 0 {
		m.lyricsTop = 0
	}
	if m.lyricsTop > maxTop {
		m.lyricsTop = maxTop
	}
}

func (m *Model) moveLyricsScroll(delta, visible, total int) {
	if delta == 0 {
		m.ensureLyricsViewport(visible, total)
		return
	}
	m.lyricsTop += delta
	m.ensureLyricsViewport(visible, total)
}

func upNextVisibleItems(queueHeight int) int {
	v := (queueHeight - 2) / 2
	if v < 1 {
		v = 1
	}
	return v
}

func lyricsVisibleItems(queueHeight int) int {
	v := queueHeight - 2
	if v < 1 {
		v = 1
	}
	return v
}
