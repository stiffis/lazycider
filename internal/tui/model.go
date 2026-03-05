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

	centerTitle    string
	centerSongs    []centerSongRow
	centerSelected int
	centerTop      int

	playlistIDByName   map[string]string
	playlistURLByName  map[string]string
	playlistCache      map[string][]centerSongRow
	activePlaylistName string

	track    string
	artist   string
	album    string
	playing  bool
	progress float64
	current  string
	total    string
	volume   int
	shuffle  bool
	repeat   int
	trackID  string

	rightPanelMode RightPanelMode
	lyricsText     string
	lyricsErr      string

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
		state:             StateNormal,
		focus:             PanelCenter,
		leftModules:       seedLeftModules(),
		leftSelected:      0,
		leftTop:           0,
		centerTitle:       "Center Content",
		centerSongs:       []centerSongRow{{Title: "Select a playlist", Artist: "Left panel", Duration: ""}},
		centerSelected:    0,
		centerTop:         0,
		playlistIDByName:  make(map[string]string),
		playlistURLByName: make(map[string]string),
		playlistCache:     make(map[string][]centerSongRow),
		track:             "Sailing",
		artist:            "Christopher Cross",
		album:             "Christopher Cross",
		playing:           true,
		progress:          0.55,
		current:           "2:34",
		total:             "4:41",
		volume:            80,
		shuffle:           false,
		repeat:            0,
		rightPanelMode:    RightPanelCover,
		upNext:            seedUpNext(),
		upNextSelected:    0,
		upNextTop:         0,
		cider:             client,
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

func upNextVisibleItems(queueHeight int) int {
	v := (queueHeight - 2) / 2
	if v < 1 {
		v = 1
	}
	return v
}

func seedUpNext() []upNextRow {
	return []upNextRow{
		{Title: "You're My Heart, You're My Soul", Subtitle: "Modern Talking — 80s 100 Hits"},
		{Title: "Listen To Your Heart", Subtitle: "Roxette — Look Sharp!"},
		{Title: "All Out of Love", Subtitle: "Air Supply — 80s 100 Hits"},
		{Title: "Una Lady Como Tu", Subtitle: "Manuel Turizo — Una Lady Como Tu"},
		{Title: "Can't Fight This Feeling", Subtitle: "REO Speedwagon — 80s 100 Hits"},
		{Title: "Escucha a tu Corazon", Subtitle: "Laura Pausini — Lo mejor"},
		{Title: "Donde esta el amor", Subtitle: "Pablo Alboran — Baladas"},
		{Title: "Hard to Say I'm Sorry", Subtitle: "Peter Cetera — Glory of Love"},
		{Title: "Animals", Subtitle: "Maroon 5 — V (Deluxe)"},
		{Title: "How Am I Supposed to Live", Subtitle: "Michael Bolton — 80s 100 Hits"},
		{Title: "Lejos de Ti", Subtitle: "Pelo D'Ambrosio — Lejos de ti"},
		{Title: "Caribbean Queen", Subtitle: "Billy Ocean — 80s 100 Hits"},
		{Title: "Yo Quisiera", Subtitle: "Reik — Reik"},
		{Title: "Run To You", Subtitle: "Bryan Adams — Reckless"},
		{Title: "Take On Me", Subtitle: "a-ha — Hunting High and Low"},
		{Title: "The Promise", Subtitle: "When In Rome — Greatest Hits"},
		{Title: "Every Time You Go Away", Subtitle: "Paul Young — The Secret of Association"},
		{Title: "Broken Wings", Subtitle: "Mr. Mister — Welcome to the Real World"},
		{Title: "Drive", Subtitle: "The Cars — Heartbeat City"},
		{Title: "Time After Time", Subtitle: "Cyndi Lauper — She's So Unusual"},
		{Title: "True", Subtitle: "Spandau Ballet — True"},
		{Title: "Against All Odds", Subtitle: "Phil Collins — Hits"},
		{Title: "One Last Cry", Subtitle: "Brian McKnight — Brian McKnight"},
		{Title: "Un-Break My Heart", Subtitle: "Toni Braxton — Secrets"},
		{Title: "End of the Road", Subtitle: "Boyz II Men — Cooleyhighharmony"},
	}
}
