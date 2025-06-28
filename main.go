package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const gap = "\n\n"

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
	client *http.Client
	width  int
	height int

	currentMode   mode
	currentMethod httpMethod
	url           textinput.Model
	responseView  viewport.Model
}

func initialModel() model {
	url := textinput.New()
	url.Prompt = ""
	return model{
		url:    url,
		client: http.DefaultClient,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var reqCmd, respCmd tea.Cmd
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.responseView = viewport.New(msg.Width-2, msg.Height-3-lipgloss.Height(m.renderUrlView()))
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.url.Blur()
			m.currentMode = modeOverview
		case "enter":
			var resp *http.Response
			var err error
			switch m.currentMethod {
			case httpMethodGet:
				resp, err = m.client.Get(m.url.Value())
			case httpMethodPost:
				resp, err = m.client.Post(m.url.Value(), "application/json", bytes.NewReader([]byte("{}")))
			}
			if err != nil {
				break
			}
			defer resp.Body.Close()
			res, err := io.ReadAll(resp.Body)
			if err != nil {
				break
			}
			m.responseView.SetContent(string(res))
		}
	}
	return m, tea.Batch(reqCmd, respCmd)
}

func (m model) View() string {
	return fmt.Sprintf("%s%s%s", m.renderUrlView(), gap, m.renderResponseView())
}

func (m model) renderUrlView() string {
	return lipgloss.NewStyle().Width(m.width - 2).Border(lipgloss.NormalBorder()).Render(m.url.View())
}

func (m model) renderResponseView() string {
	return lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Render(m.responseView.View())
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	p.Run()
}
