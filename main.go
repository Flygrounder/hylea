package main

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/charmbracelet/bubbles/stopwatch"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type mode int

const (
	modeOverview mode = iota
	modeUrl
	modeResponse
)

type httpMethod int

func (h httpMethod) toString() string {
	switch h {
	case httpMethodGet:
		return "GET"
	case httpMethodPost:
		return "POST"
	}
	panic("unknown method")
}

const (
	httpMethodGet httpMethod = iota
	httpMethodPost
)

type model struct {
	client        *http.Client
	width         int
	height        int
	currentMode   mode
	currentMethod httpMethod
	url           textinput.Model
	responseView  viewport.Model
	elapsed       stopwatch.Model
	requestTag    int
}

type responseMessage struct {
	tag      int
	response string
}

func initialModel() model {
	url := textinput.New()
	url.Prompt = ""
	return model{
		url:     url,
		client:  http.DefaultClient,
		elapsed: stopwatch.NewWithInterval(time.Millisecond),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var reqCmd, respCmd, httpCmd, timeCmd, elapsedCmd tea.Cmd
	m.elapsed, elapsedCmd = m.elapsed.Update(msg)
	switch m.currentMode {
	case modeOverview:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "u":
				m.currentMode = modeUrl
				m.url.Focus()
			case "r":
				m.currentMode = modeResponse
			case "m":
				m.currentMethod = (m.currentMethod + 1) % 2
			case "q":
				return m, tea.Quit
			}
		}
	case modeUrl:
		m.url, reqCmd = m.url.Update(msg)
	case modeResponse:
		m.responseView, respCmd = m.responseView.Update(msg)
	}

	switch msg := msg.(type) {
	case responseMessage:
		if m.requestTag != msg.tag {
			break
		}
		m.responseView.SetContent(msg.response)
		timeCmd = m.elapsed.Stop()
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.responseView = viewport.New(msg.Width-2, msg.Height-2-lipgloss.Height(m.renderUrlView())-lipgloss.Height(m.renderStatusBar()))
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.url.Blur()
			m.currentMode = modeOverview
		case "enter":
			m.requestTag++
			httpCmd = func() tea.Msg {
				var resp *http.Response
				var err error
				switch m.currentMethod {
				case httpMethodGet:
					resp, err = m.client.Get(m.url.Value())
				case httpMethodPost:
					resp, err = m.client.Post(m.url.Value(), "application/json", bytes.NewReader([]byte("{}")))
				}
				if err != nil {
					return nil
				}
				defer resp.Body.Close()
				res, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil
				}
				return responseMessage{
					tag:      m.requestTag,
					response: string(res),
				}
			}
			timeCmd = tea.Sequence(m.elapsed.Reset(), m.elapsed.Start())
		}
	}
	return m, tea.Batch(reqCmd, respCmd, httpCmd, elapsedCmd, timeCmd)
}

func (m model) View() string {
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
	return style.Render(m.currentMethod.toString())
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
	return style.Render(m.elapsed.View())
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	p.Run()
}
