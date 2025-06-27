package main

import (
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
	modeOverview = iota
	modeUrl
	modeResponse
)

type model struct {
	client *http.Client

	currentMode  mode
	url          textinput.Model
	responseView viewport.Model
}

func initialModel() model {
	url := textinput.New()
	url.Prompt = "GET > "

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
	case modeUrl:
		m.url, reqCmd = m.url.Update(msg)
	case modeResponse:
		m.responseView, respCmd = m.responseView.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.responseView = viewport.New(msg.Width, msg.Height-lipgloss.Height(m.url.View())-lipgloss.Height(gap))
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "u":
			if m.currentMode == modeOverview {
				m.currentMode = modeUrl
				m.url.Focus()
			}
		case "r":
			if m.currentMode == modeOverview {
				m.currentMode = modeResponse
			}
		case "esc":
			m.url.Blur()
			m.currentMode = modeOverview
		case "enter":
			resp, err := m.client.Get(m.url.Value())
			if err != nil {
				break
			}
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
	return fmt.Sprintf("%s%s%s", m.url.View(), gap, m.responseView.View())
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	p.Run()
}
