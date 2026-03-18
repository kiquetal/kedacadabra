package main

import (
	"fmt"
	"strconv"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type mode int

const (
	modeRelative mode = iota
	modeAbsolute
)

// Model is the Bubble Tea application state.
type Model struct {
	cfg     Config
	appIdx  int // current app index
	mode    mode
	focus   int
	status  Status
	spinner spinner.Model
	help    help.Model
	keys    keyMap
	msg     string
	msgErr  bool
	loading bool

	relInputs [2]textinput.Model
	absInputs [6]textinput.Model
}

func (m Model) app() AppConfig { return m.cfg.Apps[m.appIdx] }

// Messages
type statusMsg Status
type applyDoneMsg struct{ err error }
type clearMsgMsg struct{}

func newInput(placeholder string, charLimit int) textinput.Model {
	t := textinput.New()
	t.Placeholder = placeholder
	t.CharLimit = charLimit
	t.SetWidth(10)
	return t
}

func NewModel(cfg Config) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	m := Model{
		cfg:     cfg,
		spinner: s,
		help:    help.New(),
		keys:    defaultKeyMap,
	}

	m.relInputs[0] = newInput("minutes", 5)
	m.relInputs[1] = newInput("minutes", 5)
	m.relInputs[0].Focus()

	placeholders := [6]string{"day(0-6)", "hour", "min", "day(0-6)", "hour", "min"}
	for i := range m.absInputs {
		m.absInputs[i] = newInput(placeholders[i], 2)
	}

	return m
}

func (m Model) Init() tea.Cmd {
	app := m.app()
	return func() tea.Msg { return statusMsg(FetchStatus(app)) }
}

func (m Model) fetchCmd() tea.Cmd {
	app := m.app()
	return func() tea.Msg { return statusMsg(FetchStatus(app)) }
}

func (m Model) applyCmd(pauseCron, resumeCron string) tea.Cmd {
	app := m.app()
	return func() tea.Msg {
		return applyDoneMsg{err: ApplyCronSchedule(app, pauseCron, resumeCron)}
	}
}

func clearMsgAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return clearMsgMsg{} })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case statusMsg:
		m.status = Status(msg)
		m.loading = false
		return m, nil

	case applyDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.msg = fmt.Sprintf("✗ %v", msg.err)
			m.msgErr = true
		} else {
			m.msg = "✓ Applied successfully"
			m.msgErr = false
		}
		return m, tea.Batch(m.fetchCmd(), clearMsgAfter(5*time.Second))

	case clearMsgMsg:
		m.msg = ""
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		if m.loading {
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Refresh):
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.fetchCmd())

		case key.Matches(msg, m.keys.NextApp):
			m.appIdx = (m.appIdx + 1) % len(m.cfg.Apps)
			m.loading = true
			m.msg = ""
			return m, tea.Batch(m.spinner.Tick, m.fetchCmd())

		case key.Matches(msg, m.keys.PrevApp):
			m.appIdx = (m.appIdx - 1 + len(m.cfg.Apps)) % len(m.cfg.Apps)
			m.loading = true
			m.msg = ""
			return m, tea.Batch(m.spinner.Tick, m.fetchCmd())

		case key.Matches(msg, m.keys.ModeRel):
			m.mode = modeRelative
			m.blurAll()
			m.focus = 0
			m.relInputs[0].Focus()
			return m, nil

		case key.Matches(msg, m.keys.ModeAbs):
			m.mode = modeAbsolute
			m.blurAll()
			m.focus = 0
			m.absInputs[0].Focus()
			return m, nil

		case key.Matches(msg, m.keys.Tab):
			m.advanceFocus(1)
			return m, nil

		case key.Matches(msg, m.keys.ShiftTab):
			m.advanceFocus(-1)
			return m, nil

		case key.Matches(msg, m.keys.Apply):
			return m.handleApply()
		}
	}

	cmd := m.updateFocusedInput(msg)
	return m, cmd
}

func (m *Model) handleApply() (tea.Model, tea.Cmd) {
	var pauseCron, resumeCron string

	if m.mode == modeRelative {
		pMin, err1 := strconv.Atoi(m.relInputs[0].Value())
		rMin, err2 := strconv.Atoi(m.relInputs[1].Value())
		if err1 != nil || err2 != nil {
			m.msg = "✗ Enter valid numbers"
			m.msgErr = true
			return m, clearMsgAfter(3 * time.Second)
		}
		pauseCron = RelativeToCron(pMin)
		resumeCron = RelativeToCron(rMin)
	} else {
		vals := make([]int, 6)
		for i, inp := range m.absInputs {
			v, err := strconv.Atoi(inp.Value())
			if err != nil {
				m.msg = "✗ Enter valid numbers in all fields"
				m.msgErr = true
				return m, clearMsgAfter(3 * time.Second)
			}
			vals[i] = v
		}
		pauseCron = AbsoluteToCron(vals[0], vals[1], vals[2])
		resumeCron = AbsoluteToCron(vals[3], vals[4], vals[5])
	}

	m.loading = true
	m.msg = ""
	return m, tea.Batch(m.spinner.Tick, m.applyCmd(pauseCron, resumeCron))
}

func (m *Model) blurAll() {
	for i := range m.relInputs {
		m.relInputs[i].Blur()
	}
	for i := range m.absInputs {
		m.absInputs[i].Blur()
	}
}

func (m *Model) advanceFocus(dir int) {
	m.blurAll()
	n := m.inputCount()
	m.focus = (m.focus + dir + n) % n
	if m.mode == modeRelative {
		m.relInputs[m.focus].Focus()
	} else {
		m.absInputs[m.focus].Focus()
	}
}

func (m Model) inputCount() int {
	if m.mode == modeRelative {
		return len(m.relInputs)
	}
	return len(m.absInputs)
}

func (m *Model) updateFocusedInput(msg tea.Msg) tea.Cmd {
	if m.mode == modeRelative {
		var cmd tea.Cmd
		m.relInputs[m.focus], cmd = m.relInputs[m.focus].Update(msg)
		return cmd
	}
	var cmd tea.Cmd
	m.absInputs[m.focus], cmd = m.absInputs[m.focus].Update(msg)
	return cmd
}

// Key map
type keyMap struct {
	Quit     key.Binding
	Refresh  key.Binding
	NextApp  key.Binding
	PrevApp  key.Binding
	ModeRel  key.Binding
	ModeAbs  key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
	Apply    key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.NextApp, k.PrevApp, k.ModeRel, k.ModeAbs, k.Tab, k.Apply, k.Refresh, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

var defaultKeyMap = keyMap{
	Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	NextApp:  key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "next app")),
	PrevApp:  key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "prev app")),
	ModeRel:  key.NewBinding(key.WithKeys("f1"), key.WithHelp("F1", "relative")),
	ModeAbs:  key.NewBinding(key.WithKeys("f2"), key.WithHelp("F2", "absolute")),
	Tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
	ShiftTab: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("s-tab", "prev field")),
	Apply:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply")),
}
