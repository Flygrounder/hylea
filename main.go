package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type item string

func (i item) FilterValue() string {
	return string(i)
}

var _ list.Item = item("")

type itemDelegate struct{}

func (i itemDelegate) Height() int {
	return 1
}

func (i itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it := listItem.(item)
	var res string
	if m.Index() == index {
		res = ">" + string(it)
	} else {
		res = string(it)
	}
	_, _ = w.Write([]byte(res))
}

func (i itemDelegate) Spacing() int {
	return 0
}

func (i itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

var _ list.ItemDelegate = itemDelegate{}

type mode int

const (
	modeOverview mode = iota
	modeUrl
	modeResponse
	modeMethod
)

type model struct {
	client       *http.Client
	width        int
	height       int
	currentMode  mode
	url          textinput.Model
	responseView viewport.Model
	timer        requestTimer
	methodView   list.Model
}

type requestTimer struct {
	isActive     bool
	requestTag   int
	lastStart    time.Time
	lastDuration time.Duration
}

type responseMessage struct {
	tag      int
	response string
	err      error
}

func initialModel() model {
	url := textinput.New()
	url.Prompt = ""
	methodView := list.New([]list.Item{
		item("GET"),
		item("POST"),
	}, itemDelegate{}, 0, 0)
	methodView.Title = "HTTP Method"
	methodView.Select(0)
	methodView.DisableQuitKeybindings()
	return model{
		url:        url,
		client:     http.DefaultClient,
		methodView: methodView,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var handlerCmd, elapsedCmd tea.Cmd

	switch m.currentMode {
	case modeOverview:
		handlerCmd = m.handleMessageInOverviewMode(msg)
	case modeUrl:
		handlerCmd = m.handleMessageInUrlMode(msg)
	case modeResponse:
		handlerCmd = m.handleMessageInResponseMode(msg)
	case modeMethod:
		handlerCmd = m.handleMessageInMethodMode(msg)
	}

	globalCmd := m.handleGlobalEvents(msg)

	return m, tea.Batch(handlerCmd, elapsedCmd, globalCmd)
}

func (m *model) handleGlobalEvents(msg tea.Msg) tea.Cmd {
	var timeCmd tea.Cmd
	switch msg := msg.(type) {
	case responseMessage:
		if m.timer.requestTag != msg.tag {
			break
		}
		m.timer.isActive = false
		m.timer.lastDuration = time.Since(m.timer.lastStart)
		if msg.err != nil {
			break
		}
		m.responseView.SetContent(msg.response)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.responseView.Width = msg.Width - 2
		m.responseView.Height = msg.Height - 2 - lipgloss.Height(m.renderUrlView()) - lipgloss.Height(m.renderStatusBar())
		m.methodView.SetSize(msg.Width, msg.Height)
	}
	return timeCmd
}

func (m *model) handleMessageInResponseMode(msg tea.Msg) tea.Cmd {
	if m.currentMode != modeResponse {
		panic("cannot use response mode handler in non-response mode")
	}
	var cmd tea.Cmd
	m.responseView, cmd = m.responseView.Update(msg)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.currentMode = modeOverview
		}
	}
	return cmd
}

func (m *model) handleMessageInOverviewMode(msg tea.Msg) tea.Cmd {
	if m.currentMode != modeOverview {
		panic("cannot use overview mode handler in non-overview mode")
	}
	var httpCmd, timeCmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "u":
			m.currentMode = modeUrl
			m.url.Focus()
		case "r":
			m.currentMode = modeResponse
		case "m":
			m.currentMode = modeMethod
		case "q":
			return tea.Quit
		case "esc":
			m.url.Blur()
			m.currentMode = modeOverview
		case "enter":
			httpCmd = m.startRequest()
		}
	}
	return tea.Sequence(timeCmd, httpCmd)
}

func (m *model) handleMessageInUrlMode(msg tea.Msg) tea.Cmd {
	var widgetCmd, httpCmd tea.Cmd
	m.url, widgetCmd = m.url.Update(msg)
	if m.currentMode != modeUrl {
		panic("cannot use url mode handler in non-url mode")
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.url.Blur()
			m.currentMode = modeOverview
		case "enter":
			m.url.Blur()
			m.currentMode = modeOverview
			httpCmd = m.startRequest()
		}
	}
	return tea.Batch(widgetCmd, httpCmd)
}

func (m *model) startRequest() tea.Cmd {
	m.timer.isActive = true
	m.timer.requestTag++
	m.timer.lastStart = time.Now()
	httpCmd := func() tea.Msg {
		var resp *http.Response
		var err error
		it, ok := m.methodView.SelectedItem().(item)
		if !ok {
			return responseMessage{
				tag: m.timer.requestTag,
				err: errors.New("failed to get selected item"),
			}
		}
		switch it {
		case "GET":
			resp, err = m.client.Get(m.url.Value())
		case "POST":
			resp, err = m.client.Post(m.url.Value(), "application/json", bytes.NewReader([]byte("{}")))
		}
		if err != nil {
			return responseMessage{
				tag: m.timer.requestTag,
				err: fmt.Errorf("failed to send request: %w", err),
			}
		}
		defer resp.Body.Close()
		res, err := io.ReadAll(resp.Body)
		if err != nil {
			return responseMessage{
				tag: m.timer.requestTag,
				err: fmt.Errorf("failed to read response: %w", err),
			}
		}
		return responseMessage{
			tag:      m.timer.requestTag,
			response: string(res),
		}
	}
	return httpCmd
}

func (m *model) handleMessageInMethodMode(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.methodView, cmd = m.methodView.Update(msg)
	if m.currentMode != modeMethod {
		panic("cannot use method mode handler in non-method mode")
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.currentMode = modeOverview
		}
	}
	return cmd
}

func (m model) View() string {
	if m.currentMode == modeMethod {
		return m.methodView.View()
	}
	return lipgloss.JoinVertical(lipgloss.Left, lipgloss.JoinHorizontal(lipgloss.Top, m.renderMethod(), m.renderUrlView()), m.renderResponseView(), m.renderStatusBar())
}

func (m model) renderUrlView() string {
	style := lipgloss.NewStyle().Width(m.width - 2 - lipgloss.Width(m.renderMethod())).Border(lipgloss.NormalBorder())
	if m.currentMode == modeUrl {
		style = style.BorderForeground(lipgloss.Color("#ff0000"))
	}
	return style.Render(m.url.View())
}

func (m model) renderMethod() string {
	style := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Width(4)
	it, ok := m.methodView.SelectedItem().(item)
	if !ok {
		return ""
	}
	return style.Render(string(it))
}

func (m model) renderResponseView() string {
	style := lipgloss.NewStyle().Border(lipgloss.NormalBorder())
	if m.currentMode == modeResponse {
		style = style.BorderForeground(lipgloss.Color("#ff0000"))
	}
	return style.Render(m.responseView.View())
}

func (m model) renderStatusBar() string {
	style := lipgloss.NewStyle().Width(m.width - 2).Border(lipgloss.NormalBorder())
	return style.Render(m.renderTimer())
}

func (m model) renderTimer() string {
	var duration time.Duration
	if m.timer.isActive {
		duration = time.Since(m.timer.lastStart)
	} else {
		duration = m.timer.lastDuration
	}
	return duration.Round(time.Millisecond).String()
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	p.Run()
}
