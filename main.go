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
	modeOverview = iota
	modeUrl
	modeResponse
)

type model struct {
	client *http.Client

	currentMode  mode
	methodIndex  int
	url          textinput.Model
	responseView viewport.Model
}

func initialModel() model {
	return model{
		url:    textinput.New(),
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
				m.methodIndex = (m.methodIndex + 1) % 2
			}
		}
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
		case "esc":
			m.url.Blur()
			m.currentMode = modeOverview
		case "enter":
			var resp *http.Response
			var err error
			switch m.methodIndex {
			case 0:
				resp, err = m.client.Get(m.url.Value())
			case 1:
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
	var method string
	switch m.methodIndex {
	case 0:
		method = "GET"
	case 1:
		method = "POST"
	}
	m.url.Prompt = fmt.Sprintf("%s > ", method)
	return m, tea.Batch(reqCmd, respCmd)
}

func (m model) View() string {
	return fmt.Sprintf("%s%s%s", m.url.View(), gap, m.responseView.View())
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	p.Run()
}
