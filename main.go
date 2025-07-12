package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/flygrounder/hylea/widgets/method"
)

type mode int

const (
	modeOverview mode = iota
	modeUrl
	modeResponse
	modeMethod
	modeRequest
)

type model struct {
	dimensions   modelDimensions
	client       *http.Client
	method       method.Model
	currentMode  mode
	url          textinput.Model
	requestView  textarea.Model
	responseView viewport.Model
	timer        requestTimer
}

type modelDimensions struct {
	width  int
	height int
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
	return model{
		url:         url,
		client:      http.DefaultClient,
		method:      method.New(),
		requestView: textarea.New(),
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
		if !m.method.Focused() {
			m.currentMode = modeOverview
		}
	case modeRequest:
		handlerCmd = m.handleMessageInRequestMode(msg)
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
		m.responseView.SetContent(prettify(msg.response))
	case tea.WindowSizeMsg:
		m.dimensions = modelDimensions{
			width:  msg.Width,
			height: msg.Height,
		}
		m.requestView.SetWidth(msg.Width/2 - 2)
		m.requestView.SetHeight(msg.Height - 2 - lipgloss.Height(m.renderUrlView(m.dimensions)))

		m.responseView.Width = msg.Width/2 - 2
		m.responseView.Height = msg.Height - 2 - lipgloss.Height(m.renderStatusBar(m.dimensions))
	}
	return timeCmd
}

func (m *model) handleMessageInRequestMode(msg tea.Msg) tea.Cmd {
	if m.currentMode != modeRequest {
		panic("cannot use request mode handler in non-request mode")
	}
	var cmd tea.Cmd
	m.requestView, cmd = m.requestView.Update(msg)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.currentMode = modeOverview
		}
	}
	return cmd
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
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "u":
			m.currentMode = modeUrl
			cmd = m.url.Focus()
		case "r":
			m.currentMode = modeResponse
		case "m":
			m.currentMode = modeMethod
			cmd = m.method.Focus()
		case "q":
			return tea.Quit
		case "b":
			m.currentMode = modeRequest
			cmd = m.requestView.Focus()
		case "enter":
			cmd = m.startRequest()
		}
	}
	return cmd
}

func (m *model) handleMessageInUrlMode(msg tea.Msg) tea.Cmd {
	if m.currentMode != modeUrl {
		panic("cannot use url mode handler in non-url mode")
	}
	var widgetCmd, httpCmd tea.Cmd
	m.url, widgetCmd = m.url.Update(msg)
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
		curMethod := m.method.Value()
		switch curMethod {
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
	if m.currentMode != modeMethod {
		panic("cannot use method mode handler in non-method mode")
	}
	var cmd tea.Cmd
	m.method, cmd = m.method.Update(msg)
	return cmd
}

func (m model) View() string {
	method := m.method.View()
	url := m.renderUrlView(modelDimensions{
		width:  m.dimensions.width/2 - lipgloss.Width(method),
		height: m.dimensions.height,
	})
	requestPanel := lipgloss.JoinVertical(lipgloss.Left, lipgloss.JoinHorizontal(lipgloss.Top, method, url), m.renderRequestView())
	responsePanel := lipgloss.JoinVertical(lipgloss.Left, m.renderResponseView(), m.renderStatusBar(modelDimensions{
		width:  m.dimensions.width / 2,
		height: m.dimensions.height,
	}))
	return lipgloss.JoinHorizontal(lipgloss.Top, requestPanel, responsePanel)
}

func (m model) renderUrlView(dimensions modelDimensions) string {
	style := lipgloss.NewStyle().Width(dimensions.width - 2).Border(lipgloss.NormalBorder())
	if m.currentMode == modeUrl {
		style = style.BorderForeground(lipgloss.Color("#ff0000"))
	}
	return style.Render(m.url.View())
}

func (m model) renderRequestView() string {
	style := lipgloss.NewStyle().Border(lipgloss.NormalBorder())
	if m.currentMode == modeRequest {
		style = style.BorderForeground(lipgloss.Color("#ff0000"))
	}
	return style.Render(m.requestView.View())
}

func (m model) renderResponseView() string {
	style := lipgloss.NewStyle().Border(lipgloss.NormalBorder())
	if m.currentMode == modeResponse {
		style = style.BorderForeground(lipgloss.Color("#ff0000"))
	}
	return style.Render(m.responseView.View())
}

func (m model) renderStatusBar(dimensions modelDimensions) string {
	style := lipgloss.NewStyle().Width(dimensions.width - 2).Border(lipgloss.NormalBorder())
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

func prettify(response string) string {
	var resp map[string]any
	err := json.Unmarshal([]byte(response), &resp)
	if err != nil {
		return response
	}
	t, _ := json.MarshalIndent(resp, "", "    ")
	return string(t)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	p.Run()
}
